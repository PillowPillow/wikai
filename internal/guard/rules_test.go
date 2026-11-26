package guard

import (
	"path/filepath"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/testrepo"
)

func brainConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join("..", "config", "testdata", "brain.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestClassify(t *testing.T) {
	cfg := brainConfig(t)
	cases := map[string]Rule{
		"_meta/manifest.jsonl": AppendOnly, "_meta/lint/Run 1.md": Frozen, "_meta/schema.md": Protected,
		"10-raw/web/a.md": Frozen, "10-raw/zotero/lib.md": Protected, "10-raw/AGENTS.md": None,
		"CLAUDE.md": Protected, ".claude/settings.json": Protected, "wikai.yaml": Protected,
		".git/config": Protected, "20-wiki/concepts/A.md": None, "_metadata.md": None,
		".git": Protected, "10-raw": Frozen, "10-rawx/a.md": None, "10-raw/zoteroX/a.md": Frozen,
		"20-wiki/sub/.git/HEAD": Protected,
	}
	for p, want := range cases {
		if got := Classify(cfg, p); got != want {
			t.Errorf("Classify(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestClassifyMapsThroughRoot(t *testing.T) {
	dir := t.TempDir()
	testrepo.Write(t, dir, map[string]string{"wikai.yaml": "version: 1\nroot: vault\nfrozen:\n  - path: raw\n    except: [raw/free]\nappend_only: [log.jsonl]\nprotected: [vault/CLAUDE.md]\n"})
	cfg, err := config.Load(filepath.Join(dir, "wikai.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]Rule{
		"vault/raw/a.md": Frozen, "raw/a.md": None, "vault/raw/free/a.md": None,
		"vault/log.jsonl": AppendOnly, "log.jsonl": None, "vault/CLAUDE.md": Protected,
	}
	for p, want := range cases {
		if got := Classify(cfg, p); got != want {
			t.Errorf("Classify(%q) = %v, want %v", p, got, want)
		}
	}
}
