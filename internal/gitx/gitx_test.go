package gitx

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func TestBaseRef(t *testing.T) {
	plain := t.TempDir()
	ref, warnings, err := BaseRef(plain, "main", "")
	if ref != "" || !reflect.DeepEqual(warnings, []string{"not a git repository, frozen check skipped"}) || err != nil {
		t.Errorf("plain dir: %q %q %v", ref, warnings, err)
	}

	empty := t.TempDir()
	testrepo.Git(t, empty, "init", "-q", "-b", "main")
	ref, warnings, err = BaseRef(empty, "main", "")
	if ref != "" || warnings != nil || err != nil {
		t.Errorf("no commit: %q %q %v", ref, warnings, err)
	}

	repo := testrepo.New(t, map[string]string{"a.md": "a\n"})
	ref, warnings, err = BaseRef(repo, "main", "")
	if ref != "HEAD" || !reflect.DeepEqual(warnings, []string{"no remote, base is HEAD"}) || err != nil {
		t.Errorf("no remote: %q %q %v", ref, warnings, err)
	}
	ref, warnings, err = BaseRef(repo, "main", "abc")
	if ref != "abc" || warnings != nil || err != nil {
		t.Errorf("flag: %q %q %v", ref, warnings, err)
	}

	testrepo.Git(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	_, _, err = BaseRef(repo, "main", "")
	if err == nil || !strings.Contains(err.Error(), "no merge-base with origin/main") {
		t.Errorf("unfetched remote: %v", err)
	}

	src := testrepo.New(t, map[string]string{"a.md": "a\n"})
	parent := t.TempDir()
	bare := filepath.Join(parent, "remote.git")
	work := filepath.Join(parent, "work")
	testrepo.Git(t, parent, "clone", "-q", "--bare", src, bare)
	testrepo.Git(t, parent, "clone", "-q", bare, work)
	testrepo.Write(t, work, map[string]string{"b.md": "b\n"})
	testrepo.Git(t, work, "add", "-A")
	testrepo.Git(t, work, "commit", "-q", "-m", "local")
	want := testrepo.Git(t, work, "rev-parse", "origin/main")
	ref, warnings, err = BaseRef(work, "main", "")
	if ref != want || warnings != nil || err != nil {
		t.Errorf("clone: %q %q %v, want %q", ref, warnings, err, want)
	}
}

func TestLsFilesOddNames(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"dir/é à.md": "x\n"})
	testrepo.Write(t, dir, map[string]string{"new file.md": "y\n"})
	testrepo.Git(t, dir, "add", "new file.md")
	got, err := LsFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"dir/é à.md", "new file.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLsFilesFromSubfolder(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"top.md": "x\n", "sub/a.md": "a\n"})
	got, err := LsFiles(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDiffAndShow(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"a.md": "old\n"})
	testrepo.Write(t, dir, map[string]string{"a.md": "new\n", "b.md": "b\n"})
	testrepo.Git(t, dir, "add", "-A")
	changes, err := Diff(dir, "--cached")
	if err != nil {
		t.Fatal(err)
	}
	if want := []Change{{"M", "a.md"}, {"A", "b.md"}}; !reflect.DeepEqual(changes, want) {
		t.Errorf("diff: got %v, want %v", changes, want)
	}
	if got, err := Show(dir, "HEAD", "a.md"); err != nil || string(got) != "old\n" {
		t.Errorf("show HEAD: %q %v", got, err)
	}
	if got, err := Show(dir, "", "a.md"); err != nil || string(got) != "new\n" {
		t.Errorf("show index: %q %v", got, err)
	}
}

func TestShowAndLsTreeFromSubfolder(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"sub/a.md": "a\n", "sub/x/b.md": "b\n", "top.md": "t\n"})
	sub := filepath.Join(dir, "sub")
	if got, err := Show(sub, "HEAD", "a.md"); err != nil || string(got) != "a\n" {
		t.Errorf("show: %q %v", got, err)
	}
	got, err := LsTree(sub, "HEAD", "x")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"x/b.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ls-tree: got %q, want %q", got, want)
	}
	if _, err := Show(sub, "HEAD", "missing.md"); err == nil {
		t.Error("show of a missing file: no error")
	}
}

func TestMvAndMergeBase(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"a.md": "a\n"})
	head := testrepo.Git(t, dir, "rev-parse", "HEAD")
	if err := Mv(dir, "a.md", "b b.md"); err != nil {
		t.Fatal(err)
	}
	changes, err := Diff(dir, "--cached")
	if err != nil {
		t.Fatal(err)
	}
	if want := []Change{{"D", "a.md"}, {"A", "b b.md"}}; !reflect.DeepEqual(changes, want) {
		t.Errorf("diff after mv: got %v", changes)
	}
	if got, err := MergeBase(dir, "HEAD", "main"); err != nil || got != head {
		t.Errorf("merge-base: %q %v", got, err)
	}
	if !IsRepo(dir) || !HasCommit(dir) || IsRepo(t.TempDir()) {
		t.Error("IsRepo / HasCommit")
	}
}
