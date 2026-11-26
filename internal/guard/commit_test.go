package guard

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func vaultRepo(t *testing.T) string {
	t.Helper()
	cfg, err := os.ReadFile(filepath.Join("..", "config", "testdata", "brain.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return testrepo.New(t, map[string]string{
		"wikai.yaml":            string(cfg),
		"10-raw/web/a.md":       "a\n",
		"10-raw/zotero/lib.md":  "lib\n",
		"_meta/manifest.jsonl":  "L1\n",
		"_meta/schema.md":       "schema\n",
		"20-wiki/concepts/A.md": "A\n",
	})
}

type change func(t *testing.T, dir string)

func write(name, content string) change {
	return func(t *testing.T, dir string) { testrepo.Write(t, dir, map[string]string{name: content}) }
}

func gitDo(args ...string) change {
	return func(t *testing.T, dir string) { testrepo.Git(t, dir, args...) }
}

var commitCases = []struct {
	name  string
	edits []change
	agent bool
	want  []Violation
}{
	{"new raw file", []change{write("10-raw/web/b.md", "b\n")}, false, nil},
	{"modified raw file", []change{write("10-raw/web/a.md", "a2\n")}, false, []Violation{{"M", "10-raw/web/a.md", "frozen file"}}},
	{"deleted raw file", []change{gitDo("rm", "-q", "10-raw/web/a.md")}, false, []Violation{{"D", "10-raw/web/a.md", "frozen file"}}},
	{"moved raw file", []change{gitDo("mv", "10-raw/web/a.md", "10-raw/web/z.md")}, false, []Violation{{"D", "10-raw/web/a.md", "frozen file"}}},
	{"zotero", []change{write("10-raw/zotero/lib.md", "lib2\n")}, false, nil},
	{"zotero agent", []change{write("10-raw/zotero/lib.md", "lib2\n")}, true, []Violation{{"M", "10-raw/zotero/lib.md", "protected path"}}},
	{"manifest grown", []change{write("_meta/manifest.jsonl", "L1\nL2\n")}, true, nil},
	{"manifest rewritten", []change{write("_meta/manifest.jsonl", "X1\nL2\n")}, false, []Violation{{"M", "_meta/manifest.jsonl", "append-only file: existing lines changed"}}},
	{"manifest deleted", []change{gitDo("rm", "-q", "_meta/manifest.jsonl")}, false, []Violation{{"D", "_meta/manifest.jsonl", "append-only file: existing lines changed"}}},
	{"schema", []change{write("_meta/schema.md", "schema2\n")}, false, nil},
	{"schema agent", []change{write("_meta/schema.md", "schema2\n")}, true, []Violation{{"M", "_meta/schema.md", "protected path"}}},
	{"new raw file agent", []change{write("10-raw/web/b.md", "b\n")}, true, nil},
	{"case variant agent", []change{write("agents.md", "x\n")}, true, []Violation{{"A", "agents.md", "protected path"}}},
	{"case variant frozen", []change{write("10-RAW/web/b.md", "b\n"), write("10-raw/web/a.md", "a2\n")}, false, []Violation{{"M", "10-raw/web/a.md", "frozen file"}}},
	{"sorted", []change{write("_meta/schema.md", "s\n"), write("10-raw/web/a.md", "a2\n"), write("10-raw/zotero/lib.md", "l\n")}, true, []Violation{
		{"M", "10-raw/web/a.md", "frozen file"}, {"M", "10-raw/zotero/lib.md", "protected path"}, {"M", "_meta/schema.md", "protected path"},
	}},
}

func TestCommitStaged(t *testing.T) {
	for _, c := range commitCases {
		t.Run(c.name, func(t *testing.T) {
			dir := vaultRepo(t)
			for _, e := range c.edits {
				e(t, dir)
			}
			testrepo.Git(t, dir, "add", "-A")
			got, err := Commit(brainConfig(t), dir, true, "", "", c.agent)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Commit = %v, %v; want %v", got, err, c.want)
			}
		})
	}
}

func TestCommitStagedReadsTheIndex(t *testing.T) {
	dir := vaultRepo(t)
	testrepo.Write(t, dir, map[string]string{"_meta/manifest.jsonl": "X\n"})
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Write(t, dir, map[string]string{"_meta/manifest.jsonl": "L1\nL2\n"})
	got, err := Commit(brainConfig(t), dir, true, "", "", false)
	if err != nil || len(got) != 1 {
		t.Fatalf("Commit = %v, %v", got, err)
	}
}

func TestCommitRange(t *testing.T) {
	for _, c := range commitCases {
		t.Run(c.name, func(t *testing.T) {
			dir := vaultRepo(t)
			testrepo.Git(t, dir, "checkout", "-q", "-b", "work")
			for _, e := range c.edits {
				e(t, dir)
			}
			testrepo.Git(t, dir, "add", "-A")
			testrepo.Git(t, dir, "commit", "-q", "-m", "change")
			testrepo.Git(t, dir, "checkout", "-q", "main")
			testrepo.Write(t, dir, map[string]string{"20-wiki/concepts/B.md": "B\n"})
			testrepo.Git(t, dir, "add", "-A")
			testrepo.Git(t, dir, "commit", "-q", "-m", "main moves on")
			got, err := Commit(brainConfig(t), dir, false, "main", "work", c.agent)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Commit = %v, %v; want %v", got, err, c.want)
			}
		})
	}
}

func TestCommitBadRef(t *testing.T) {
	dir := vaultRepo(t)
	if _, err := Commit(brainConfig(t), dir, false, "main", "nope", false); err == nil {
		t.Fatal("want an error")
	}
}
