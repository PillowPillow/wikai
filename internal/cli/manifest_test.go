package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

const manifestYAML = "version: 1\nroot: .\nmanifest:\n  file: _meta/manifest.jsonl\n  pipeline: ingest-v2\n  statuses: [integrated, no_material, error, superseded]\n"

func manifestHome(t *testing.T, yaml string) string {
	t.Helper()
	return testrepo.New(t, map[string]string{
		"wikai.yaml":  yaml,
		"10-raw/n.md": "Some  source\n\ntext.\n",
	})
}

func TestManifestCLI(t *testing.T) {
	dir := manifestHome(t, manifestYAML)
	for name, args := range map[string][]string{
		"no status":      {"manifest", "append", "10-raw/n.md"},
		"unknown status": {"manifest", "append", "10-raw/n.md", "--status", "nope"},
	} {
		code, out, errOut := run(t, dir, args...)
		if code != 2 || out != "" || errOut == "" {
			t.Fatalf("%s: code=%d out=%q err=%q", name, code, out, errOut)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "_meta", "manifest.jsonl")); err == nil {
		t.Fatal("manifest written on usage error")
	}

	for _, args := range [][]string{
		{"manifest", "append", "10-raw/n.md", "--status", "integrated", "--pages", "A", "B"},
		{"manifest", "append", "--pages", "wiki/concepts/X.md", "wiki/concepts/Y.md", "10-raw/n.md", "--status", "integrated"},
	} {
		code, out, errOut := run(t, dir, args...)
		if code != 2 || out != "" || errOut == "" {
			t.Fatalf("%v: code=%d out=%q err=%q", args, code, out, errOut)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "_meta", "manifest.jsonl")); err == nil {
		t.Fatal("manifest written on extra positional")
	}

	code, out, errOut := run(t, dir, "manifest", "append", "10-raw/n.md", "--status", "integrated", "--pages", "A", "--pages", "B", "--note", "ok")
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	want := `{"ts":"2026-01-02T03:04:05Z","raw":"10-raw/n.md","sha256":"`
	if !strings.HasPrefix(out, want) || !strings.HasSuffix(out, `","status":"integrated","pipeline":"ingest-v2","pages":["A","B"],"note":"ok"}`+"\n") {
		t.Fatalf("out=%q", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "_meta", "manifest.jsonl"))
	if err != nil || string(data) != out {
		t.Fatalf("file=%q err=%v", data, err)
	}

	code, out2, errOut := run(t, dir, "manifest", "append", "10-raw/n.md", "--status", "integrated", "--pages", "A,B", "--note", "ok")
	if code != 0 || errOut != "" || !strings.Contains(out2, `"pages":["A","B"]`) {
		t.Fatalf("code=%d out=%q err=%q", code, out2, errOut)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "_meta", "manifest.jsonl")); string(data) != out+out2 {
		t.Fatalf("file=%q", data)
	}

	code, hashOut, _ := run(t, dir, "manifest", "hash", "10-raw/n.md")
	if code != 0 || len(hashOut) != 65 || !strings.Contains(out, hashOut[:64]) {
		t.Fatalf("code=%d hash=%q", code, hashOut)
	}

	code, out, errOut = run(t, filepath.Join(dir, "10-raw"), "manifest", "status", "n.md")
	if code != 0 || out != "unchanged integrated 10-raw/n.md\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}

	if err := os.WriteFile(filepath.Join(dir, "10-raw", "n.md"), []byte("different\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = run(t, filepath.Join(dir, "10-raw"), "manifest", "status", "n.md")
	if code != 0 || out != "modified\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestManifestCLIStatusNew(t *testing.T) {
	dir := manifestHome(t, manifestYAML)
	code, out, _ := run(t, dir, "manifest", "status", "10-raw/n.md")
	if code != 0 || out != "new\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestManifestCLINoBlock(t *testing.T) {
	dir := manifestHome(t, "version: 1\nroot: .\n")
	for _, sub := range []string{"hash", "status"} {
		code, out, errOut := run(t, dir, "manifest", sub, "10-raw/n.md")
		if code != 2 || out != "" || !strings.Contains(errOut, "manifest") {
			t.Fatalf("%s: code=%d out=%q err=%q", sub, code, out, errOut)
		}
	}
	code, _, _ := run(t, dir, "manifest", "append", "10-raw/n.md", "--status", "integrated")
	if code != 2 {
		t.Fatalf("append code=%d", code)
	}
}
