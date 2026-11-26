package check_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PillowPillow/wikai/internal/cli"
	"github.com/PillowPillow/wikai/internal/testrepo"
	"github.com/PillowPillow/wikai/internal/wiki"
)

func TestFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}
	var dirs int
	for _, e := range entries {
		if e.IsDir() {
			dirs++
		}
	}
	if dirs != 53 {
		t.Errorf("fixture folder holds %d fixtures, want 53", dirs)
	}
	ported, err := os.ReadFile("testdata/ported.txt")
	if err != nil {
		t.Fatal(err)
	}
	names := strings.Fields(string(ported))
	if len(names) != 29 {
		t.Errorf("ported.txt lists %d names, want 29", len(names))
	}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join("testdata/fixtures", name)); err != nil {
			t.Errorf("ported fixture %s: %v", name, err)
		}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			runFixture(t, filepath.Join("testdata/fixtures", e.Name()))
		})
	}
}

func runFixture(t *testing.T, fixture string) {
	dir := testrepo.New(t, readTree(t, filepath.Join(fixture, "base")))
	clearWorktree(t, dir)
	testrepo.Write(t, dir, readTree(t, filepath.Join(fixture, "tree")))
	cfg, err := os.ReadFile(filepath.Join(fixture, "wikai.yaml"))
	if os.IsNotExist(err) {
		cfg, err = os.ReadFile("testdata/home.yaml")
	}
	if err != nil {
		t.Fatal(err)
	}
	testrepo.Write(t, dir, map[string]string{"wikai.yaml": string(cfg)})
	testrepo.Git(t, dir, "add", "-A")

	var stdout, stderr bytes.Buffer
	code := cli.Execute([]string{"check", "--json"}, cli.Env{
		Dir: dir, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Getenv: func(string) string { return "" }, Now: time.Now,
	})
	var got []wiki.Finding
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("code=%d stdout=%q stderr=%q: %v", code, stdout.String(), stderr.String(), err)
	}
	data, err := os.ReadFile(filepath.Join(fixture, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected []wiki.Finding
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	wantCode := 1
	if len(expected) == 0 {
		wantCode = 0
	}
	if code != wantCode {
		t.Errorf("code=%d, want %d; stderr=%q", code, wantCode, stderr.String())
	}
	remaining := got
	for _, want := range expected {
		var matches []int
		for i, f := range remaining {
			if f.Check == want.Check && f.Path == want.Path && strings.Contains(f.Message, want.Message) {
				matches = append(matches, i)
			}
		}
		if len(matches) != 1 {
			t.Errorf("expected %+v matches %d findings in %+v", want, len(matches), got)
			continue
		}
		remaining = append(remaining[:matches[0]:matches[0]], remaining[matches[0]+1:]...)
	}
	if len(remaining) != 0 {
		t.Errorf("unexpected findings: %+v", remaining)
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return files
}

func clearWorktree(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
}
