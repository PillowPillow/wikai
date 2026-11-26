package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/preset"
	"github.com/PillowPillow/wikai/internal/testrepo"
)

func initPreset(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	code, out, errOut := run(t, dir, "init", dir, "--preset", name)
	if code != 0 || errOut != "" || !strings.Contains(out, "created wikai.yaml\n") {
		t.Fatalf("%s: code=%d out=%q err=%q", name, code, out, errOut)
	}
	testrepo.Git(t, dir, "init", "-q", "-b", "main")
	testrepo.Git(t, dir, "add", "-A")
	return dir
}

func TestInitThenCheck(t *testing.T) {
	want := map[string]string{"home-repo": "OK: 0 pages\n", "obsidian-vault": "OK: 2 pages\n"}
	for _, name := range preset.Names() {
		dir := initPreset(t, name)
		code, out, errOut := run(t, dir, "check")
		if code != 0 || out != want[name] {
			t.Fatalf("%s: code=%d out=%q err=%q", name, code, out, errOut)
		}
	}
}

func TestInitDefaultsToCurrentFolderAndCreatesDir(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := run(t, dir, "init", "--preset", "home-repo"); code != 0 {
		t.Fatalf("%d %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "wikai.yaml")); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(t, dir, "init", "sub/base", "--preset", "obsidian-vault")
	if code != 0 || !strings.Contains(out, "created sub/base/wikai.yaml\n") {
		t.Fatalf("%d %q", code, out)
	}
}

func TestInitEveryKindCreates(t *testing.T) {
	for _, name := range preset.Names() {
		dir := initPreset(t, name)
		cfg, err := config.Load(filepath.Join(dir, "wikai.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var kinds []string
		for _, ty := range cfg.Types {
			if len(ty.Folders) == 0 {
				continue
			}
			for k := range ty.Kinds {
				kinds = append(kinds, k)
			}
		}
		slices.Sort(kinds)
		if len(kinds) == 0 {
			t.Fatalf("%s: no kind", name)
		}
		for _, k := range kinds {
			args := []string{"new", "page", k, "T-" + k, "--summary", "s"}
			if k == "repo" {
				args = append(args, "--set", "origin=git@example.com:acme/x.git")
			}
			if code, out, errOut := run(t, dir, args...); code != 0 {
				t.Fatalf("%s %s: code=%d out=%q err=%q", name, k, code, out, errOut)
			}
		}
		testrepo.Git(t, dir, "add", "-A")
		_, out, _ := run(t, dir, "check", "--json")
		var findings []struct{ Check string }
		if err := json.Unmarshal([]byte(out), &findings); err != nil {
			t.Fatalf("%s: %v: %s", name, err, out)
		}
		for _, f := range findings {
			if f.Check != "orphans" {
				t.Fatalf("%s: %s", name, out)
			}
		}
	}
}

func TestInitRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	if code, _, _ := run(t, dir, "init", "--preset", "home-repo"); code != 0 {
		t.Fatal(code)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "wikai.yaml"))
	code, _, errOut := run(t, dir, "init", "--preset", "obsidian-vault")
	after, _ := os.ReadFile(filepath.Join(dir, "wikai.yaml"))
	if code != 1 || !strings.HasPrefix(errOut, "refused: ") || string(before) != string(after) {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestInitUsage(t *testing.T) {
	for name, args := range map[string][]string{
		"no preset":      {"init"},
		"unknown preset": {"init", "--preset", "nope"},
	} {
		dir := t.TempDir()
		if code, _, errOut := run(t, dir, args...); code != 2 || !strings.HasPrefix(errOut, "error: ") {
			t.Fatalf("%s: code=%d err=%q", name, code, errOut)
		}
		if _, err := os.Stat(filepath.Join(dir, "wikai.yaml")); err == nil {
			t.Fatalf("%s: wrote files", name)
		}
	}
}
