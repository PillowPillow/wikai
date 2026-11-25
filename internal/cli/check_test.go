package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func homeTree(t *testing.T, fixture string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join("..", "check", "testdata", "fixtures", fixture, "tree")
	if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	cfg, err := fs.ReadFile(os.DirFS(filepath.Join("..", "check", "testdata")), "home.yaml")
	if err != nil {
		t.Fatal(err)
	}
	testrepo.Write(t, dir, map[string]string{"wikai.yaml": string(cfg)})
	return dir
}

func TestCheckText(t *testing.T) {
	code, out, errOut := run(t, homeTree(t, "valid"), "check")
	if code != 0 || out != "OK: 3 pages\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = run(t, homeTree(t, "frontmatter-none"), "check")
	if code != 1 || out != "frontmatter: knowledge/glossary/contract.md: no frontmatter\nFAIL: 1 finding(s)\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	if strings.Contains(errOut, "refused") {
		t.Fatalf("err=%q", errOut)
	}
}

func TestCheckJSON(t *testing.T) {
	code, out, _ := run(t, homeTree(t, "valid"), "check", "--json")
	if code != 0 || out != "[]\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, _ = run(t, homeTree(t, "frontmatter-none"), "check", "--json")
	want := "[\n  {\n    \"check\": \"frontmatter\",\n    \"path\": \"knowledge/glossary/contract.md\",\n    \"message\": \"no frontmatter\"\n  }\n]\n"
	if code != 1 || out != want {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestCheckSorted(t *testing.T) {
	dir := homeTree(t, "valid")
	testrepo.Write(t, dir, map[string]string{
		"knowledge/notes.md":      "# Notes\n",
		"knowledge/glossary/b.md": "---\ntype: term\nsources: []\naliases: []\n---\n# b\n",
		"knowledge/glossary/a.md": "# a\n",
	})
	code, out, errOut := run(t, dir, "check")
	want := "frontmatter: knowledge/glossary/a.md: no frontmatter\n" +
		"index: knowledge/glossary/a.md: page is not linked from the index\n" +
		"orphans: knowledge/glossary/a.md: no other page links to this page\n" +
		"frontmatter: knowledge/glossary/b.md: missing key: created\n" +
		"frontmatter: knowledge/glossary/b.md: missing key: updated\n" +
		"index: knowledge/glossary/b.md: page is not linked from the index\n" +
		"orphans: knowledge/glossary/b.md: no other page links to this page\n" +
		"stray: knowledge/notes.md: markdown file outside the page folders\n" +
		"FAIL: 8 finding(s)\n"
	if code != 1 || out != want {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCheckNoConfig(t *testing.T) {
	code, _, errOut := run(t, t.TempDir(), "check")
	if code != 2 || !strings.Contains(errOut, "no wikai.yaml") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCheckOutsideGit(t *testing.T) {
	code, out, errOut := run(t, homeTree(t, "valid"), "check")
	if code != 0 || out != "OK: 3 pages\n" || !strings.Contains(errOut, "warning: not a git repository") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCheckBaseFlagInRepo(t *testing.T) {
	dir := homeTree(t, "valid")
	testrepo.Git(t, dir, "init", "-q", "-b", "main")
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Git(t, dir, "commit", "-q", "-m", "init")
	code, out, errOut := run(t, dir, "check", "--base", "HEAD")
	if code != 0 || out != "OK: 3 pages\n" || errOut != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCheckBaseFlag(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":                         "version: 1\nroot: knowledge\nstray: false\nfrozen:\n  - path: log\n    checked: false\n",
		"knowledge/log/2026-01-01-init-x.md": "- 2026-01-01: created.\n",
	})
	testrepo.Write(t, dir, map[string]string{"knowledge/log/2026-01-01-init-x.md": "- 2026-01-01: rewritten.\n"})
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Git(t, dir, "commit", "-q", "-m", "modify")

	if code, out, errOut := run(t, dir, "check"); code != 0 {
		t.Fatalf("default base: code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, _ := run(t, dir, "check", "--base", "HEAD~1")
	if code != 1 || !strings.Contains(out, "frozen: knowledge/log/2026-01-01-init-x.md: frozen file modified") {
		t.Fatalf("HEAD~1: code=%d out=%q", code, out)
	}
	code, _, errOut := run(t, dir, "check", "--base", "nope")
	if code != 2 || !strings.Contains(errOut, `cannot read git ref "nope"`) {
		t.Fatalf("nope: code=%d err=%q", code, errOut)
	}
}
