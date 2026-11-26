package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func vault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg, err := os.ReadFile(filepath.Join("..", "config", "testdata", "brain.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	testrepo.Write(t, dir, map[string]string{
		"wikai.yaml":            string(cfg),
		"CLAUDE.md":             "# rules\n",
		"10-raw/web/a.md":       "a\n",
		"10-raw/zotero/lib.md":  "lib\n",
		"_meta/lint/Run 1.md":   "run 1\n",
		"_meta/manifest.jsonl":  "L1\n",
		"_meta/schema.md":       "schema\n",
		"20-wiki/concepts/A.md": "A\n",
	})
	return dir
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func event(tool, cwd, file, content string) Event {
	var ev Event
	ev.ToolName, ev.Cwd = tool, cwd
	ev.ToolInput.FilePath, ev.ToolInput.Content = file, content
	return ev
}

func TestHook(t *testing.T) {
	dir := vault(t)
	cfg := brainConfig(t)
	symlink(t, "CLAUDE.md", filepath.Join(dir, "AGENTS.md"))
	symlink(t, "../CLAUDE.md", filepath.Join(dir, "20-wiki", "x.md"))
	symlink(t, "../_meta", filepath.Join(dir, "20-wiki", "m"))
	symlink(t, "../_meta/lint", filepath.Join(dir, "20-wiki", "l"))
	symlink(t, "../_meta/new.md", filepath.Join(dir, "20-wiki", "d.md"))
	abs := func(p string) string { return filepath.Join(dir, p) }
	notebook := event("NotebookEdit", dir, "", "")
	notebook.ToolInput.NotebookPath = abs("10-raw/zotero/n.ipynb")
	cases := []struct {
		name string
		ev   Event
		deny string
	}{
		{"new raw file", event("Write", dir, abs("10-raw/web/new.md"), "x"), ""},
		{"edit raw file", event("Edit", dir, abs("10-raw/web/a.md"), ""), "10-raw/web/a.md: frozen file, existing files are never modified"},
		{"new lint report", event("Write", dir, abs("_meta/lint/Run 2.md"), "x"), ""},
		{"rewrite lint report", event("Write", dir, abs("_meta/lint/Run 1.md"), "x"), "frozen"},
		{"manifest append", event("Write", dir, abs("_meta/manifest.jsonl"), "L1\nL2\n"), ""},
		{"manifest rewrite", event("Write", dir, abs("_meta/manifest.jsonl"), "X\n"), "_meta/manifest.jsonl: append-only file, new content must start with the current content"},
		{"manifest edit", event("Edit", dir, abs("_meta/manifest.jsonl"), ""), "append-only"},
		{"protected", event("Write", dir, abs("_meta/schema.md"), "x"), "_meta/schema.md: protected path, propose the change to the human"},
		{"notebook", notebook, "protected"},
		{"outside", event("Write", dir, "/elsewhere/x.md", "x"), ""},
		{"relative", event("Write", dir, "CLAUDE.md", "x"), "CLAUDE.md: protected"},
		{"relative from subfolder", event("Write", abs("20-wiki"), "../CLAUDE.md", "x"), "CLAUDE.md: protected"},
		{"dotdot inside path", event("Write", dir, dir+"/20-wiki/../CLAUDE.md", "x"), "CLAUDE.md: protected"},
		{"symlink AGENTS.md", event("Write", dir, abs("AGENTS.md"), "x"), "protected"},
		{"symlink plain name", event("Write", dir, abs("20-wiki/x.md"), "x"), "CLAUDE.md: protected"},
		{"symlinked folder", event("Write", dir, abs("20-wiki/m/schema.md"), "x"), "_meta/schema.md: protected"},
		{"dotdot after symlink", event("Write", dir, dir+"/20-wiki/l/../schema.md", "x"), "_meta/schema.md: protected"},
		{"dangling symlink", event("Write", dir, abs("20-wiki/d.md"), "x"), "_meta/new.md: protected"},
		{"new file in missing folder", event("Write", dir, abs("20-wiki/new/deep/p.md"), "x"), ""},
		{"git folder", event("Write", dir, abs(".git/config"), "x"), "protected"},
		{"no target", event("Write", dir, "", "x"), ""},
		{"relative without cwd", event("Write", "", "20-wiki/p.md", "x"), "relative"},
		{"wiki page", event("Edit", dir, abs("20-wiki/concepts/A.md"), ""), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Hook(cfg, dir, c.ev)
			if (c.deny == "") != (got == "") || !strings.Contains(got, c.deny) {
				t.Fatalf("Hook = %q, want %q", got, c.deny)
			}
		})
	}
}

func TestHookCaseVariantOfMissingGuardedPath(t *testing.T) {
	dir := vault(t)
	cfg := brainConfig(t)
	cases := []struct{ tool, path, deny string }{
		{"Write", "agents.md", "agents.md: protected"},
		{"Write", "Agents.MD", "Agents.MD: protected"},
		{"Write", ".CLAUDE/settings.json", ".CLAUDE/settings.json: protected"},
		{"Write", "10-RAW/web/new.md", ""},
		{"Write", "10-RAW/new/new.md", ""},
	}
	for _, c := range cases {
		got := Hook(cfg, dir, event(c.tool, dir, dir+"/"+c.path, "x"))
		if (c.deny == "") != (got == "") || !strings.HasPrefix(got, c.deny) {
			t.Errorf("%s: Hook = %q, want %q", c.path, got, c.deny)
		}
	}
}

func TestHookCaseInsensitiveFilesystem(t *testing.T) {
	dir := vault(t)
	if _, err := os.Lstat(filepath.Join(dir, "claude.md")); err != nil {
		t.Skip("case-sensitive filesystem")
	}
	cfg := brainConfig(t)
	if got := Hook(cfg, dir, event("Edit", dir, dir+"/10-RAW/WEB/A.md", "")); got != "10-raw/web/a.md: frozen file, existing files are never modified" {
		t.Errorf("10-RAW/WEB/A.md: Hook = %q", got)
	}
	for _, p := range []string{"claude.md", "_META/schema.md", "_Meta/new.md", "20-WIKI/../Claude.MD"} {
		got := Hook(cfg, dir, event("Write", dir, dir+"/"+p, "x"))
		if !strings.Contains(got, "protected") {
			t.Errorf("%s: Hook = %q", p, got)
		}
	}
}

func TestHookBoundaryThroughSymlink(t *testing.T) {
	dir := vault(t)
	link := filepath.Join(t.TempDir(), "vault")
	symlink(t, dir, link)
	cfg := brainConfig(t)
	if got := Hook(cfg, link, event("Write", dir, filepath.Join(dir, "CLAUDE.md"), "x")); !strings.HasPrefix(got, "CLAUDE.md: protected") {
		t.Fatalf("Hook = %q", got)
	}
	if got := Hook(cfg, dir, event("Write", link, "CLAUDE.md", "x")); !strings.HasPrefix(got, "CLAUDE.md: protected") {
		t.Fatalf("Hook = %q", got)
	}
}
