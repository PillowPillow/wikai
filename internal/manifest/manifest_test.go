package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var golden = []struct{ in, norm, sha string }{
	{"Hello  world\n\nSecond line.\n", "Hello world Second line.",
		"411aaaca8e6dc12957cbd87d69ee340f0c2efa5859ba8a461b9a215fcb1c68cf"},
	{"\ufeffHello\r\nworld\r\n", "Hello world",
		"64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c"},
	{"---\ntype: source\ntags: []\n---\n# Title\n\nBody text.\n", "# Title Body text.",
		"5006560973a1a5576d49014be4376537585edaf531cd6745f304c351d1e6bb1f"},
	{"A quoted line ^abc-123\nAnother line. ^q2  \nNo id here ^ not-an-id\n",
		"A quoted line Another line. No id here ^ not-an-id",
		"4e68ae6304210a303457b012af36a52163086cbec0c83853bbd1e5dce150d628"},
	{"café noir fin\n", "café noir fin",
		"584e789766a0db33676704d7485a78f300eaeb80161fcb2baf09b8a3c8baa6b1"},
	{"---\na: 1\n--- trailing\nBody\n", "trailing Body",
		"d24aed7af3ed9be2ad5cafa014e43173f79e87ea1c7f80536373f126be5cc6a8"},
}

func TestNormalizeAndHash(t *testing.T) {
	for _, g := range golden {
		if got := Normalize(g.in); got != g.norm {
			t.Errorf("Normalize(%q) = %q, want %q", g.in, got, g.norm)
		}
		sum := sha256.Sum256([]byte(g.norm))
		if hex.EncodeToString(sum[:]) != g.sha {
			t.Errorf("golden sha mismatch for %q", g.in)
		}
		f := filepath.Join(t.TempDir(), "n.md")
		if err := os.WriteFile(f, []byte(g.in), 0o644); err != nil {
			t.Fatal(err)
		}
		if got, err := Hash(f); err != nil || got != g.sha {
			t.Errorf("Hash(%q) = %q, %v", g.in, got, err)
		}
	}
}

func TestNormalizeFileSeparatorIsSpace(t *testing.T) {
	if got := Normalize("a\x1cb c"); got != "a b c" {
		t.Fatalf("%q", got)
	}
}

func TestLine(t *testing.T) {
	r := Record{TS: "2026-01-02T03:04:05Z", Raw: "10-raw/web/Café <a&b>.md", SHA256: "ab", Status: "integrated", Pipeline: "ingest-v2"}
	want := `{"ts":"2026-01-02T03:04:05Z","raw":"10-raw/web/Café <a&b>.md","sha256":"ab","status":"integrated","pipeline":"ingest-v2","pages":[],"note":""}`
	if got := r.Line(); got != want {
		t.Fatalf("got %s", got)
	}
	r.Pages = []string{"A", "B"}
	if got := r.Line(); !strings.Contains(got, `"pages":["A","B"]`) {
		t.Fatalf("got %s", got)
	}
}

func TestStatus(t *testing.T) {
	dir := t.TempDir()
	mf := filepath.Join(dir, "_meta", "manifest.jsonl")
	note := filepath.Join(dir, "n.md")
	write := func(s string) {
		if err := os.WriteFile(note, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	digest := func() string {
		d, err := Hash(note)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	status := func(raw, d string) string {
		s, err := Status(mf, raw, d)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	write("one two\n")
	d := digest()
	if got := status("10-raw/n.md", d); got != "new" {
		t.Fatalf("absent: %q", got)
	}
	if err := os.MkdirAll(filepath.Dir(mf), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mf, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := status("10-raw/n.md", d); got != "new" {
		t.Fatalf("empty: %q", got)
	}
	spaced := `{"ts": "2026-01-01T00:00:00Z", "raw": "10-raw/n.md", "sha256": "` + d + `", "status": "integrated", "pipeline": "p", "pages": [], "note": ""}` + "\n\n"
	if err := os.WriteFile(mf, []byte(spaced), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := status("10-raw/n.md", d); got != "unchanged integrated 10-raw/n.md" {
		t.Fatalf("spaced: %q", got)
	}
	if err := os.Remove(mf); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(mf, Record{Raw: "10-raw/n.md", SHA256: d, Status: "integrated"}); err != nil {
		t.Fatal(err)
	}
	if got := status("10-raw/n.md", d); got != "unchanged integrated 10-raw/n.md" {
		t.Fatalf("appended: %q", got)
	}
	if got := status("10-raw/other.md", d); got != "unchanged integrated 10-raw/n.md" {
		t.Fatalf("other path: %q", got)
	}
	write("changed content\n")
	if got := status("10-raw/n.md", digest()); got != "modified" {
		t.Fatalf("modified: %q", got)
	}
	if _, err := Append(mf, Record{Raw: "10-raw/n.md", SHA256: d, Status: "superseded"}); err != nil {
		t.Fatal(err)
	}
	if got := status("10-raw/n.md", d); got != "unchanged superseded 10-raw/n.md" {
		t.Fatalf("last wins: %q", got)
	}
}

func TestStatusBadLine(t *testing.T) {
	mf := filepath.Join(t.TempDir(), "m.jsonl")
	if err := os.WriteFile(mf, []byte("{}\n\nnot json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(mf, "r", "d"); err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("err = %v", err)
	}
}

func TestAppendConcurrent(t *testing.T) {
	mf := filepath.Join(t.TempDir(), "sub", "m.jsonl")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Append(mf, Record{Raw: "r", SHA256: "x", Status: "integrated", Note: strings.Repeat("é", 5000)}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(mf)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 20 {
		t.Fatalf("%d lines", len(lines))
	}
	for _, l := range lines {
		var r Record
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAppendAfterMissingTrailingNewline(t *testing.T) {
	mf := filepath.Join(t.TempDir(), "m.jsonl")
	if err := os.WriteFile(mf, []byte(`{"raw":"a"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	line, err := Append(mf, Record{Raw: "b", SHA256: "x", Status: "integrated"})
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(mf); string(data) != `{"raw":"a"}`+"\n"+line+"\n" {
		t.Fatalf("%q", data)
	}
}
