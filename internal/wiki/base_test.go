package wiki

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/testrepo"
	"golang.org/x/text/unicode/norm"
)

func loadConfig(t *testing.T, name string) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join("..", "config", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func loadBase(t *testing.T, dir, cfgName string) *Base {
	t.Helper()
	b, err := Load(dir, loadConfig(t, cfgName))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func pagePaths(b *Base) []string {
	var out []string
	for _, p := range b.Pages {
		out = append(out, p.Path)
	}
	return out
}

var homeFiles = map[string]string{
	"knowledge/index.md":                 "# Index\n",
	"knowledge/README.md":                "readme",
	"knowledge/_skeletons/term.md":       "---\ntype: term\n---\n# {{title}}\n",
	"knowledge/glossary/contract.md":     "---\ntype: term\n---\n# Contract\n",
	"knowledge/archive/old.md":           "# Old\n",
	"knowledge/log/2026-01-01-init-x.md": "## [2026-01-01] init | x\n",
	"knowledge/notes.md":                 "stray",
	"knowledge/glossary/latin1.md":       "caf\xe9",
}

func TestLoadHome(t *testing.T) {
	b := loadBase(t, testrepo.New(t, homeFiles), "home.yaml")
	if got, want := pagePaths(b), []string{"knowledge/glossary/contract.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pages: got %q, want %q", got, want)
	}
	p := b.Pages[0]
	if p.Title != "Contract" || p.Type != "term" || p.Folder != "glossary" || p.Frozen || p.Unchecked || p.MetaErr != nil || p.RawMeta != "type: term\n" {
		t.Errorf("page: %+v", p)
	}
	if b.Page("knowledge/glossary/contract.md") != p || b.Page("knowledge/notes.md") != nil {
		t.Error("Page lookup")
	}
	wantDocs := []*Doc{
		{Path: "knowledge/archive/old.md", Text: "# Old\n", OutsideOnly: true},
		{Path: "knowledge/index.md", Text: "# Index\n"},
	}
	if !reflect.DeepEqual(b.Docs, wantDocs) {
		t.Errorf("docs: got %+v", b.Docs)
	}
	if want := []string{"knowledge/notes.md"}; !reflect.DeepEqual(b.Stray, want) {
		t.Errorf("stray: got %q", b.Stray)
	}
	if want := []Finding{{"frontmatter", "knowledge/glossary/latin1.md", "not UTF-8"}}; !reflect.DeepEqual(b.LoadFindings, want) {
		t.Errorf("load findings: got %+v", b.LoadFindings)
	}
	if b.IndexPath() != "knowledge/index.md" || b.LogPath() != "knowledge/log" || !b.Git {
		t.Errorf("IndexPath %q, LogPath %q, Git %v", b.IndexPath(), b.LogPath(), b.Git)
	}
	if !b.Files["knowledge/README.md"] || !b.Files["knowledge/_skeletons/term.md"] || len(b.Files) != len(homeFiles) {
		t.Errorf("files: %v", b.Files)
	}
}

func TestLoadVault(t *testing.T) {
	dir := t.TempDir()
	testrepo.Write(t, dir, map[string]string{
		"index.md":                 "---\ntype: meta\n---\n# Index\n",
		"log.md":                   "## log\n",
		"10-raw/a.md":              "raw",
		"10-raw/zotero/z.md":       "z",
		"00-inbox/capture.md":      "c",
		"notes.md":                 "---\n- a\n---\n",
		"50-missions/m.md":         "---\ntype: mission\n---\n",
		"_meta/skeletons/x.md":     "skeleton",
		"_meta/lint/report.md":     "ignored",
		".git/x.md":                "not walked",
		"20-wiki/concepts/Idée.md": "---\ntype: [wiki\n---\n",
	})
	b := loadBase(t, dir, "brain.yaml")
	want := []string{"00-inbox/capture.md", "10-raw/a.md", "10-raw/zotero/z.md", "20-wiki/concepts/Idée.md", "50-missions/m.md", "index.md", "notes.md"}
	if got := pagePaths(b); !reflect.DeepEqual(got, want) {
		t.Fatalf("pages: got %q, want %q", got, want)
	}
	if want := []*Doc{{Path: "log.md", Text: "## log\n"}}; !reflect.DeepEqual(b.Docs, want) {
		t.Errorf("docs: %+v", b.Docs)
	}
	if b.Stray != nil || b.LoadFindings != nil || b.Git {
		t.Errorf("stray %q, findings %+v, git %v", b.Stray, b.LoadFindings, b.Git)
	}
	if p := b.Page("00-inbox/capture.md"); !p.Unchecked || p.Title != "capture" || p.Meta != nil {
		t.Errorf("inbox: %+v", p)
	}
	if p := b.Page("10-raw/a.md"); !p.Frozen || p.Folder != "" {
		t.Errorf("raw: %+v", p)
	}
	if p := b.Page("10-raw/zotero/z.md"); p.Frozen || p.Folder != "" {
		t.Errorf("frozen except follows no frozen rule at load: %+v", p)
	}
	if p := b.Page("50-missions/m.md"); p.Folder != "50-missions" || p.Type != "mission" {
		t.Errorf("mission: %+v", p)
	}
	if p := b.Page("20-wiki/concepts/Idée.md"); p.MetaErr == nil || p.Meta != nil || p.Folder != "20-wiki" || p.Title != "Idée" {
		t.Errorf("invalid yaml: %+v", p)
	}
	if p := b.Page("notes.md"); p.MetaErr == nil || p.MetaErr.Error() != "not a mapping" {
		t.Errorf("not a mapping: %+v", p)
	}
	if b.IndexPath() != "index.md" || b.LogPath() != "log.md" {
		t.Errorf("IndexPath %q, LogPath %q", b.IndexPath(), b.LogPath())
	}
	if b.Files[".git/x.md"] || !b.Files["_meta/lint/report.md"] {
		t.Errorf("files: %v", b.Files)
	}
}

func TestResolveMarkdown(t *testing.T) {
	dir := testrepo.New(t, homeFiles)
	testrepo.Write(t, dir, map[string]string{"knowledge/glossary/untracked.png": "x"})
	b := loadBase(t, dir, "home.yaml")
	from := "knowledge/glossary/contract.md"
	cases := []struct {
		link   Link
		target string
		r      Resolution
	}{
		{Link{Target: "../index.md"}, "knowledge/index.md", Resolved},
		{Link{Target: "../../../x.md"}, "", Outside},
		{Link{Target: "Contract.md"}, "knowledge/glossary/Contract.md", NotFound},
		{Link{Target: "", Anchor: "top", HasAnchor: true}, from, Resolved},
		{Link{Target: "untracked.png"}, "knowledge/glossary/untracked.png", NotFound},
		{Link{Target: "https://example.com", External: true}, "", Skipped},
		{Link{Target: "../README.md"}, "knowledge/README.md", Resolved},
	}
	for _, c := range cases {
		target, r := b.Resolve(from, c.link)
		if r != c.r || (c.target != "" && target != c.target) {
			t.Errorf("%q: got %q %v, want %q %v", c.link.Target, target, r, c.target, c.r)
		}
	}
}

func TestResolveWikilink(t *testing.T) {
	dir := t.TempDir()
	testrepo.Write(t, dir, map[string]string{
		norm.NFD.String("20-wiki/concepts/Café.md"): "# Café\n",
		"20-wiki/a/Note.md":                         "# Note\n",
		"x/20-wiki/a/Note.md":                       "# Note\n",
		"img/pic.png":                               "png",
	})
	b := loadBase(t, dir, "brain.yaml")
	from := "20-wiki/a/Note.md"
	cases := []struct {
		target, want string
		r            Resolution
	}{
		{"café", "20-wiki/concepts/Café.md", Resolved},
		{"Note", "20-wiki/a/Note.md", Resolved},
		{"a/Note", "20-wiki/a/Note.md", Resolved},
		{"x/20-wiki/a/Note.md", "x/20-wiki/a/Note.md", Resolved},
		{"pic.png", "img/pic.png", Resolved},
		{"", from, Resolved},
		{"Missing", "", NotFound},
	}
	for _, c := range cases {
		got, r := b.Resolve(from, Link{Target: c.target})
		if got != c.want || r != c.r {
			t.Errorf("%q: got %q %v, want %q %v", c.target, got, r, c.want, c.r)
		}
	}
	if b.Page("20-wiki/concepts/Café.md") == nil {
		t.Error("NFD file not loaded under its NFC path")
	}
}

func TestRel(t *testing.T) {
	dir := testrepo.New(t, homeFiles)
	b := loadBase(t, dir, "home.yaml")
	cwd := filepath.Join(dir, "knowledge", "glossary")
	if got, err := b.Rel(cwd, "contract.md"); err != nil || got != "knowledge/glossary/contract.md" {
		t.Errorf("got %q %v", got, err)
	}
	if got, err := b.Rel(cwd, filepath.Join(dir, "knowledge", "index.md")); err != nil || got != "knowledge/index.md" {
		t.Errorf("absolute: got %q %v", got, err)
	}
	_, err := b.Rel(cwd, "../../../x.md")
	if err == nil || exit.Code(err) != 2 {
		t.Errorf("outside: %v", err)
	}
}
