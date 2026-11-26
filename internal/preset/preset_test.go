package preset

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/guard"
	"github.com/PillowPillow/wikai/internal/marker"
)

var day = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

func TestNames(t *testing.T) {
	if got := Names(); !slices.Equal(got, []string{"home-repo", "obsidian-vault"}) {
		t.Fatalf("%v", got)
	}
}

func TestPresetsLoad(t *testing.T) {
	for _, name := range Names() {
		dir := t.TempDir()
		if _, err := Write(name, dir, day); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := config.Load(filepath.Join(dir, "wikai.yaml")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestWriteFillsDateOutsideSkeletons(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write("obsidian-vault", dir, day); err != nil {
		t.Fatal(err)
	}
	index, _ := os.ReadFile(filepath.Join(dir, "index.md"))
	skel, _ := os.ReadFile(filepath.Join(dir, "_meta/skeletons/concept.md"))
	if !strings.Contains(string(index), "created: 2026-01-02\n") || strings.Contains(string(index), "{{") {
		t.Fatalf("%q", index)
	}
	if !strings.Contains(string(skel), "{{title}}") {
		t.Fatalf("%q", skel)
	}
}

func TestWriteGitkeeps(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write("home-repo", dir, day); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"glossary", "repos", "domains", "decisions", "designs", "archive", "log"} {
		if _, err := os.Stat(filepath.Join(dir, "knowledge", f, ".gitkeep")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWriteRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wikai.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Write("home-repo", dir, day)
	if exit.Code(err) != 1 {
		t.Fatalf("%v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "wikai.yaml")); string(got) != "x" {
		t.Fatalf("%q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "knowledge")); err == nil {
		t.Fatal("wrote files despite refusal")
	}
}

func TestWriteUnknownPreset(t *testing.T) {
	if _, err := Write("nope", t.TempDir(), day); exit.Code(err) != 2 {
		t.Fatalf("%v", err)
	}
}

func TestWriteKeepsExistingFilesAndReturnsRelativePaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "log.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := Write("obsidian-vault", dir, day)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "log.md")); string(got) != "mine" {
		t.Fatalf("%q", got)
	}
	if slices.Contains(created, "log.md") || !slices.Contains(created, "wikai.yaml") || !slices.Contains(created, "_meta/skeletons/concept.md") {
		t.Fatalf("%v", created)
	}
}

func TestPresetsProtectTheGuardControls(t *testing.T) {
	for _, name := range Names() {
		dir := t.TempDir()
		if _, err := Write(name, dir, day); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		cfg, err := config.Load(filepath.Join(dir, "wikai.yaml"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, p := range []string{"wikai.yaml", ".claude/settings.json"} {
			if got := guard.Classify(cfg, p); got != guard.Protected {
				t.Errorf("%s: %s is %v, want protected", name, p, got)
			}
		}
	}
}

func TestPresetSkeletonsUseOnlySupportedTokens(t *testing.T) {
	for _, name := range Names() {
		dir := t.TempDir()
		if _, err := Write(name, dir, day); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(filepath.Join(dir, "wikai.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		skels, _ := filepath.Glob(filepath.Join(dir, filepath.FromSlash(cfg.RootPath(cfg.Skeletons)), "*.md"))
		if len(skels) == 0 {
			t.Fatalf("%s: no skeleton", name)
		}
		for _, s := range skels {
			data, _ := os.ReadFile(s)
			if errs := marker.Check(string(data)); len(errs) != 0 {
				t.Errorf("%s: %v", s, errs)
			}
		}
	}
}
