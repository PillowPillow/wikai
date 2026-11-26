package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func newHome(t *testing.T) string {
	t.Helper()
	dir := homeTree(t, "valid")
	testrepo.Write(t, dir, map[string]string{
		"knowledge/index.md":           "# Index\n\n## glossary\n\nNo page.\n\n## repos\n\nNo page.\n",
		"knowledge/_skeletons/term.md": "---\ntype: term\ncreated:\nupdated:\nsources: []\naliases: []\n---\n\n# {{title}}\n\n## Definition\n",
	})
	return dir
}

func TestNewPageCLI(t *testing.T) {
	dir := newHome(t)
	code, out, errOut := run(t, filepath.Join(dir, "knowledge", "glossary"), "new", "page", "term", "Offer", "--summary", "x", "--date", "2026-01-02")
	lines := strings.Split(out, "\n")
	if code != 0 || lines[0] != "created knowledge/glossary/offer.md and its line in knowledge/index.md" ||
		lines[1] != "next: fill the sections, link the page from another page, add the log entry (wikai new log)" || errOut != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	data, err := os.ReadFile(filepath.Join(dir, "knowledge/glossary/offer.md"))
	if err != nil || !strings.Contains(string(data), "created: 2026-01-02\n") {
		t.Fatalf("%q %v", data, err)
	}
}

func TestNewPageCLIDefaultDate(t *testing.T) {
	dir := newHome(t)
	code, _, errOut := run(t, dir, "new", "page", "term", "Offer", "--summary", "x")
	data, _ := os.ReadFile(filepath.Join(dir, "knowledge/glossary/offer.md"))
	if code != 0 || !strings.Contains(string(data), "created: 2026-01-02\n") {
		t.Fatalf("code=%d err=%q data=%q", code, errOut, data)
	}
}

func TestNewPageCLINoIndexLine(t *testing.T) {
	dir := newHome(t)
	testrepo.Write(t, dir, map[string]string{
		"wikai.yaml":                    "version: 1\nroot: knowledge\ntypes:\n  term:\n    folders: [glossary]\n  other:\n    folders: [elsewhere]\nindex:\n  file: index.md\n  include: [glossary]\n  groups: folder\nskeletons: _skeletons\n",
		"knowledge/_skeletons/other.md": "# {{title}}\n",
	})
	code, out, errOut := run(t, dir, "new", "page", "other", "Thing", "--summary", "x")
	if code != 0 || strings.Split(out, "\n")[0] != "created knowledge/elsewhere/thing.md" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestNewPageCLIUsage(t *testing.T) {
	for name, args := range map[string][]string{
		"no summary":    {"new", "page", "term", "Offer"},
		"bad date":      {"new", "page", "term", "Offer", "--summary", "x", "--date", "2026-02-30"},
		"short date":    {"new", "page", "term", "Offer", "--summary", "x", "--date", "2026-2-3"},
		"unknown kind":  {"new", "page", "nope", "Offer", "--summary", "x"},
		"set novalue":   {"new", "page", "term", "Offer", "--summary", "x", "--set", "novalue"},
		"missing title": {"new", "page", "term", "--summary", "x"},
		"title leaves":  {"new", "page", "term", "../x", "--summary", "x", "--slug", "../x"},
	} {
		dir := newHome(t)
		code, out, errOut := run(t, dir, args...)
		if code != 2 {
			t.Errorf("%s: code=%d out=%q err=%q", name, code, out, errOut)
		}
		if _, err := os.Stat(filepath.Join(dir, "knowledge/glossary/offer.md")); err == nil {
			t.Errorf("%s: page written", name)
		}
	}
}

func TestNewPageCLITitleLeavesFolder(t *testing.T) {
	dir := newHome(t)
	testrepo.Write(t, dir, map[string]string{"wikai.yaml": strings.Replace(readFile(t, filepath.Join(dir, "wikai.yaml")), `name: "adr-{date:compact}-{slug}"`, `name: "{title}"`, 1) + "\n"})
	testrepo.Write(t, dir, map[string]string{"knowledge/_skeletons/decision.md": "# {{title}}\n"})
	before := listFiles(t, dir)
	index := readFile(t, filepath.Join(dir, "knowledge/index.md"))
	for _, title := range []string{"a/b", "../x", `a\b`} {
		code, out, errOut := run(t, dir, "new", "page", "decision", title, "--summary", "x")
		if code != 2 || out != "" {
			t.Errorf("%q: code=%d out=%q err=%q", title, code, out, errOut)
		}
	}
	if after := listFiles(t, dir); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Fatalf("files changed:\nbefore %v\nafter %v", before, after)
	}
	if got := readFile(t, filepath.Join(dir, "knowledge/index.md")); got != index {
		t.Fatalf("index changed: %q", got)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNewPageCLIRefused(t *testing.T) {
	dir := newHome(t)
	code, _, errOut := run(t, dir, "new", "page", "term", "contract", "--summary", "x")
	if code != 1 || !strings.HasPrefix(errOut, "refused: ") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestNewLogCLI(t *testing.T) {
	dir := newHome(t)
	code, out, errOut := run(t, filepath.Join(dir, "knowledge", "glossary"), "new", "log", "ingest", "Billing", "--created", "offer.md", "--modified", "../index.md", "--note", "n", "--date", "2026-01-02")
	if code != 0 || out != "created knowledge/log/2026-01-02-ingest-billing.md\n" || errOut != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	data, err := os.ReadFile(filepath.Join(dir, "knowledge/log/2026-01-02-ingest-billing.md"))
	want := "## [2026-01-02] ingest | Billing\n- created: knowledge/glossary/offer.md\n- modified: knowledge/index.md\n- note: n\n"
	if err != nil || string(data) != want {
		t.Fatalf("%q %v", data, err)
	}
	if code, _, _ := run(t, dir, "new", "log", "ingest", "Billing", "--note", "n", "--date", "2026-01-02"); code != 1 {
		t.Fatalf("code=%d", code)
	}
	if code, _, _ := run(t, dir, "new", "log", "ingest", "X", "--created", "../../outside.md"); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestNewPageCLIRefusesWikilinkBreakingName(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := run(t, dir, "init", "--preset", "obsidian-vault"); code != 0 {
		t.Fatalf("init: %q", errOut)
	}
	before := listFiles(t, dir)
	index := readFile(t, filepath.Join(dir, "index.md"))
	code, out, errOut := run(t, dir, "new", "page", "concept", "C# basics", "--summary", "x")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "refused: ") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	if after := listFiles(t, dir); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Fatalf("files changed:\nbefore %v\nafter %v", before, after)
	}
	if got := readFile(t, filepath.Join(dir, "index.md")); got != index {
		t.Fatalf("index changed: %q", got)
	}
}
