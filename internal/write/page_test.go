package write

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/testrepo"
	"github.com/PillowPillow/wikai/internal/wiki"
)

const homeIndex = "# Index\n\n## glossary\n\n- [contract](glossary/contract.md) — commercial commitment of a customer\n\n## repos\n\n- [example-backend](repos/example-backend.md) — backend repository of the acme platform\n\n## domains\n\n- [api-contract](domains/api-contract.md) — public API domain exposed to partners\n\n## decisions\n\nNo page.\n\n## designs\n\nNo page.\n"

var homeSkeletons = map[string]string{
	"_skeletons/term.md":     "---\ntype: term\ncreated:\nupdated:\nsources: []\naliases: []\n---\n\n# {{title}}\n\n## Definition\n",
	"_skeletons/decision.md": "---\ntype: decision\ncreated:\nupdated:\nsources: []\nsupersedes:\nsuperseded-by:\n---\n\n# {{title}}\n\n## Statement\n\n## Reason\n",
	"_skeletons/fact.md":     "---\ntype: decision\ncreated:\nupdated:\nsources: []\nsupersedes:\nsuperseded-by:\n---\n\n# {{title}}\n\n## Statement\n",
	"_skeletons/design.md":   "---\ntype: design\ncreated:\nupdated:\nsources: []\n---\n\n# {{title}}\n\n## Design\n",
	"_skeletons/repo.md":     "---\ntype: repo\ncreated:\nupdated:\nsources: []\norigin: {{origin}}\naliases: []\n---\n\n# {{title}}\n\n## Role\n",
}

func homeBase(t *testing.T) (*wiki.Base, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "check", "testdata", "fixtures", "valid", "tree"))); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join("..", "check", "testdata", "home.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"wikai.yaml": string(cfg), "knowledge/index.md": homeIndex}
	for k, v := range homeSkeletons {
		files["knowledge/"+k] = v
	}
	testrepo.Write(t, dir, files)
	return loadBase(t, dir)
}

func loadBase(t *testing.T, dir string) (*wiki.Base, string) {
	t.Helper()
	cfg, boundary, err := config.Locate(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := wiki.Load(boundary, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return b, dir
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	return err == nil
}

var jan2 = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

func TestNewPageHome(t *testing.T) {
	b, dir := homeBase(t)
	p, err := NewPage(b, PageOpts{Kind: "decision", Title: "Use one binary", Summary: " One tool. ", Date: jan2})
	if err != nil {
		t.Fatal(err)
	}
	if p != "knowledge/decisions/adr-20260102-use-one-binary.md" {
		t.Fatalf("path = %q", p)
	}
	want := "---\ntype: decision\ncreated: 2026-01-02\nupdated: 2026-01-02\nsources: []\nsupersedes:\nsuperseded-by:\n---\n\n# Use one binary\n\n## Statement\n\nNot documented.\n\n## Reason\n\nNot documented.\n"
	if got := read(t, dir, p); got != want {
		t.Fatalf("content = %q", got)
	}
	idx := read(t, dir, "knowledge/index.md")
	if !strings.Contains(idx, "## decisions\n\n- [Use one binary](decisions/adr-20260102-use-one-binary.md) — One tool.\n\n## designs") {
		t.Fatalf("index = %q", idx)
	}
	if strings.Contains(idx, "## decisions\n\nNo page.") || !strings.Contains(idx, "## designs\n\nNo page.\n") {
		t.Fatalf("index = %q", idx)
	}

	for kind, want := range map[string]string{
		"fact":   "knowledge/decisions/fact-20260102-t-fact.md",
		"design": "knowledge/designs/2026-01-02-t-design.md",
	} {
		b, _ = loadBase(t, dir)
		if p, err := NewPage(b, PageOpts{Kind: kind, Title: "T " + kind, Summary: "s", Date: jan2}); err != nil || p != want {
			t.Fatalf("%s: %q %v", kind, p, err)
		}
	}
	b, _ = loadBase(t, dir)
	if p, err := NewPage(b, PageOpts{Kind: "term", Title: "Other", Summary: "s", Slug: "x", Date: jan2}); err != nil || p != "knowledge/glossary/x.md" {
		t.Fatalf("%q %v", p, err)
	}
}

func TestNewPageSet(t *testing.T) {
	b, dir := homeBase(t)
	testrepo.Write(t, dir, map[string]string{"knowledge/_skeletons/repo.md": "---\ntype: repo\ncreated:\nupdated:\nsources: []\norigin: {{origin}}\naliases: []\n---\n\n# {{title}}\n\nOther {{other}} {{summary}}\n"})
	p, err := NewPage(b, PageOpts{Kind: "repo", Title: "Tool", Summary: "{{title}}", Date: jan2, Set: map[string]string{"origin": "git@example.com:acme/example-backend.git"}})
	if err != nil {
		t.Fatal(err)
	}
	got := read(t, dir, p)
	if !strings.Contains(got, "origin: git@example.com:acme/example-backend.git\n") || !strings.Contains(got, "Other {{other}} {{title}}\n") {
		t.Fatalf("content = %q", got)
	}
}

func TestNewPageSetDoesNotAddFrontmatterKey(t *testing.T) {
	b, dir := homeBase(t)
	p, err := NewPage(b, PageOpts{Kind: "term", Title: "Offer", Summary: "s", Date: jan2, Set: map[string]string{"color": "red"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, p); strings.Contains(got, "color") {
		t.Fatalf("content = %q", got)
	}
}

func TestIndexGroup(t *testing.T) {
	b, dir := homeBase(t)
	testrepo.Write(t, dir, map[string]string{"knowledge/index.md": "# Index\n\n## glossary\n\nNo page.\n\n## repos\n\nNo page.\n"})
	if _, err := NewPage(b, PageOpts{Kind: "term", Title: "Mid", Summary: "s", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "knowledge/index.md"); got != "# Index\n\n## glossary\n\n- [Mid](glossary/mid.md) — s\n\n## repos\n\nNo page.\n" {
		t.Fatalf("index = %q", got)
	}
	for _, title := range []string{"Zed", "Alpha", "Mike"} {
		b, _ = loadBase(t, dir)
		if _, err := NewPage(b, PageOpts{Kind: "term", Title: title, Summary: "s", Date: jan2}); err != nil {
			t.Fatal(err)
		}
	}
	want := "## glossary\n\n- [Alpha](glossary/alpha.md) — s\n- [Mid](glossary/mid.md) — s\n- [Mike](glossary/mike.md) — s\n- [Zed](glossary/zed.md) — s\n\n## repos"
	if got := read(t, dir, "knowledge/index.md"); !strings.Contains(got, want) {
		t.Fatalf("index = %q", got)
	}
}

func TestIndexGroupMissing(t *testing.T) {
	b, dir := homeBase(t)
	testrepo.Write(t, dir, map[string]string{"knowledge/index.md": "# Index\n\n## glossary\n\nNo page.\n"})
	_, err := NewPage(b, PageOpts{Kind: "repo", Title: "Tool", Summary: "s", Date: jan2})
	if err == nil || exit.Code(err) != 2 || !strings.Contains(err.Error(), "## repos") {
		t.Fatalf("err = %v", err)
	}
	if exists(dir, "knowledge/repos/tool.md") {
		t.Fatal("page written")
	}
}

func TestIndexRemoveResort(t *testing.T) {
	b, _ := homeBase(t)
	idx := "## glossary\n\n- [b](glossary/b.md) — x\n- [a](glossary/a.md) — y\n\n## repos\n\nNo page.\n"
	got, err := IndexResort(b, idx, "knowledge/glossary/a.md")
	if err != nil || got != "## glossary\n\n- [a](glossary/a.md) — y\n- [b](glossary/b.md) — x\n\n## repos\n\nNo page.\n" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = IndexRemove(b, idx, "knowledge/glossary/b.md")
	if err != nil || got != "## glossary\n\n- [a](glossary/a.md) — y\n\n## repos\n\nNo page.\n" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = IndexRemove(b, got, "knowledge/glossary/a.md")
	if err != nil || got != "## glossary\n\nNo page.\n\n## repos\n\nNo page.\n" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestLinkTo(t *testing.T) {
	b, _ := homeBase(t)
	if got := LinkTo(b, "knowledge/glossary/a.md", "knowledge/repos/my repo.md", "T [x]"); got != `[T \[x\]](../repos/my%20repo.md)` {
		t.Fatalf("got %q", got)
	}
	if got := LinkTo(b, "knowledge/index.md", "knowledge/repos/r.md", "R"); got != "[R](repos/r.md)" {
		t.Fatalf("got %q", got)
	}
}

func TestNewPageRefused(t *testing.T) {
	b, dir := homeBase(t)
	testrepo.Write(t, dir, map[string]string{"knowledge/glossary/other.md": "# Other\n"})
	_, err := NewPage(b, PageOpts{Kind: "term", Title: "contract", Summary: "s", Slug: "fresh", Date: jan2})
	if exit.Code(err) != 1 || !strings.Contains(err.Error(), "knowledge/glossary/contract.md") {
		t.Fatalf("err = %v", err)
	}
	_, err = NewPage(b, PageOpts{Kind: "term", Title: "CONTRACT", Summary: "s", Slug: "fresh", Date: jan2})
	if exit.Code(err) != 1 || !strings.Contains(err.Error(), `"CONTRACT" is already the title or an alias of knowledge/glossary/contract.md: update that page`) {
		t.Fatalf("err = %v", err)
	}
	_, err = NewPage(b, PageOpts{Kind: "term", Title: "Agreement", Summary: "s", Date: jan2})
	if exit.Code(err) != 1 || !strings.Contains(err.Error(), "contract.md") {
		t.Fatalf("alias: %v", err)
	}
	_, err = NewPage(b, PageOpts{Kind: "term", Title: "Whatever", Summary: "s", Slug: "other", Date: jan2})
	if exit.Code(err) != 1 || !strings.Contains(err.Error(), "knowledge/glossary/other.md exists") {
		t.Fatalf("exists: %v", err)
	}
	if got := read(t, dir, "knowledge/index.md"); got != homeIndex {
		t.Fatalf("index changed: %q", got)
	}
}

func TestNewPageUsage(t *testing.T) {
	b, dir := homeBase(t)
	for name, o := range map[string]PageOpts{
		"unknown kind":  {Kind: "nope", Title: "T", Summary: "s"},
		"empty slug":    {Kind: "term", Title: "—", Summary: "s"},
		"slash slug":    {Kind: "term", Title: "T", Slug: "a/b", Summary: "s"},
		"dotdot slug":   {Kind: "term", Title: "T", Slug: "..", Summary: "s"},
		"dot slug":      {Kind: "term", Title: "T", Slug: ".", Summary: "s"},
		"backslash":     {Kind: "term", Title: "T", Slug: `a\b`, Summary: "s"},
		"empty title":   {Kind: "term", Title: "  ", Summary: "s"},
		"title newline": {Kind: "term", Title: "a\nb", Summary: "s"},
		"set newline":   {Kind: "repo", Title: "T", Summary: "s", Set: map[string]string{"origin": "a\nb"}},
	} {
		if _, err := NewPage(b, o); err == nil || exit.Code(err) != 2 {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if got := read(t, dir, "knowledge/index.md"); got != homeIndex {
		t.Fatalf("index changed")
	}
	if exists(dir, "knowledge/glossary/..md") || exists(dir, "knowledge/glossary/t.md") {
		t.Fatal("page written")
	}
}

func TestNewPageMissingSkeleton(t *testing.T) {
	b, dir := homeBase(t)
	if err := os.Remove(filepath.Join(dir, "knowledge/_skeletons/term.md")); err != nil {
		t.Fatal(err)
	}
	_, err := NewPage(b, PageOpts{Kind: "term", Title: "T", Summary: "s"})
	if err == nil || exit.Code(err) != 2 || !strings.Contains(err.Error(), "term.md is missing") {
		t.Fatalf("err = %v", err)
	}
}

func TestNewPageKindWithoutFolder(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":            "version: 1\nroot: kb\ntypes:\n  meta: {}\nindex:\n  file: index.md\n",
		"kb/_skeletons/meta.md": "# {{title}}\n",
		"kb/index.md":           "# Index\n",
	})
	b, _ := loadBase(t, dir)
	_, err := NewPage(b, PageOpts{Kind: "meta", Title: "T", Summary: "s"})
	if err == nil || exit.Code(err) != 2 || err.Error() != "kind meta has no folder" {
		t.Fatalf("err = %v", err)
	}
}

func TestNewPageNoGroupsFlatIndex(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":            "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes]\nindex:\n  file: index.md\n",
		"kb/_skeletons/note.md": "# {{title}}\n\n## Body\n",
		"kb/index.md":           "# Index\n\n- [b](notes/b.md) — x\n",
		"kb/notes/b.md":         "# b\n",
	})
	b, _ := loadBase(t, dir)
	if _, err := NewPage(b, PageOpts{Kind: "note", Title: "A", Summary: "s"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "kb/index.md"); got != "# Index\n\n- [A](notes/a.md) — s\n- [b](notes/b.md) — x\n" {
		t.Fatalf("index = %q", got)
	}
}

func TestNewPageNoIndexLineWhenIndexOff(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":            "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes]\n",
		"kb/_skeletons/note.md": "# {{title}}\n",
	})
	b, _ := loadBase(t, dir)
	p, err := NewPage(b, PageOpts{Kind: "note", Title: "A", Summary: "s"})
	if err != nil || Indexed(b, p) || !exists(dir, p) {
		t.Fatalf("%q %v", p, err)
	}
}

const vaultIndex = "# Index\n\n## Concepts et comparaisons (`20-wiki/concepts/`)\n\nAucune page.\n\n## Résumés de sources (`20-wiki/resumes/`)\n\n- [[Résumé - b]] — x\n- [[Résumé - Z]] — x\n"

func vaultBase(t *testing.T) (*wiki.Base, string) {
	t.Helper()
	cfg, err := os.ReadFile(filepath.Join("..", "config", "testdata", "brain.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":                     string(cfg),
		"index.md":                       vaultIndex,
		"_meta/skeletons/wiki-entite.md": "---\ntype: wiki\nkind:\nstatus: draft\nentity_kind:\ncreated: {{date}}\nupdated: {{date}}\nreviewed:\nsources: []\n---\n\n# {{title}}\n",
		"_meta/skeletons/wiki-resume.md": "---\ntype: wiki\nkind:\nstatus: draft\ncreated: {{date}}\nupdated:\nreviewed:\nsources: []\n---\n\n# {{title}}\n",
		"_meta/skeletons/mission.md":     "---\ntype: mission\ncreated:\nupdated:\nsources: []\n---\n",
		"20-wiki/resumes/Résumé - b.md":  "---\ntype: wiki\n---\n",
		"20-wiki/resumes/Résumé - Z.md":  "---\ntype: wiki\n---\n",
	})
	return loadBase(t, dir)
}

func TestNewPageVault(t *testing.T) {
	b, dir := vaultBase(t)
	p, err := NewPage(b, PageOpts{Kind: "entity", Title: "Acme Tool", Summary: "A tool.", Date: jan2})
	if err != nil || p != "20-wiki/entites/Acme Tool.md" {
		t.Fatalf("%q %v", p, err)
	}
	got := read(t, dir, p)
	if !strings.Contains(got, "\nkind: entity\n") || !strings.Contains(got, "\nreviewed:\n") || !strings.Contains(got, "created: 2026-01-02\n") {
		t.Fatalf("content = %q", got)
	}
	if Indexed(b, p) {
		t.Fatal("entites has no group: no index line expected")
	}

	b, _ = loadBase(t, dir)
	p, err = NewPage(b, PageOpts{Kind: "summary", Title: "Talk", Summary: "About a talk.", Date: jan2})
	if err != nil || p != "20-wiki/resumes/Résumé - Talk.md" {
		t.Fatalf("%q %v", p, err)
	}
	if !strings.Contains(read(t, dir, p), "\nkind: summary\n") {
		t.Fatalf("content = %q", read(t, dir, p))
	}
	want := "## Résumés de sources (`20-wiki/resumes/`)\n\n- [[Résumé - b]] — x\n- [[Résumé - Talk]] — About a talk.\n- [[Résumé - Z]] — x\n"
	if got := read(t, dir, "index.md"); !strings.HasSuffix(got, want) || !strings.Contains(got, "Aucune page.") {
		t.Fatalf("index = %q", got)
	}
}

func TestNewPageVaultOutsideInclude(t *testing.T) {
	b, dir := vaultBase(t)
	p, err := NewPage(b, PageOpts{Kind: "mission", Title: "Alpha", Summary: "s", Date: jan2})
	if err != nil || p != "50-missions/alpha.md" {
		t.Fatalf("%q %v", p, err)
	}
	if got := read(t, dir, "index.md"); got != vaultIndex {
		t.Fatalf("index changed: %q", got)
	}
}

func TestNewPageVaultTitleLeavesFolder(t *testing.T) {
	b, dir := vaultBase(t)
	for _, title := range []string{"a/b", "../x", `a\b`, ".."} {
		_, err := NewPage(b, PageOpts{Kind: "entity", Title: title, Summary: "s", Date: jan2})
		if err == nil || exit.Code(err) != 2 {
			t.Errorf("%q: err = %v", title, err)
		}
	}
	for _, rel := range []string{"20-wiki/entites/a", "20-wiki/x.md", "20-wiki/entites"} {
		if exists(dir, rel) {
			t.Errorf("%s written", rel)
		}
	}
	if got := read(t, dir, "index.md"); got != vaultIndex {
		t.Fatalf("index changed")
	}
}

func TestNewPageFlatIndexRuns(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":            "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes]\nindex:\n  file: index.md\n",
		"kb/_skeletons/note.md": "# {{title}}\n",
		"kb/index.md":           "# Index\n\n## Late\n\n- [m](notes/m.md) — m\n- [z](notes/z.md) — z\n\n## Early\n\n- [b](notes/b.md) — b\n- [d](notes/d.md) — d\n",
		"kb/notes/z.md":         "# z\n",
	})
	b, _ := loadBase(t, dir)
	if _, err := NewPage(b, PageOpts{Kind: "note", Title: "C", Summary: "c", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	want := "# Index\n\n## Late\n\n- [m](notes/m.md) — m\n- [z](notes/z.md) — z\n\n## Early\n\n- [b](notes/b.md) — b\n- [C](notes/c.md) — c\n- [d](notes/d.md) — d\n"
	if got := read(t, dir, "kb/index.md"); got != want {
		t.Fatalf("index = %q", got)
	}
	b, _ = loadBase(t, dir)
	if _, err := NewPage(b, PageOpts{Kind: "note", Title: "Zz", Summary: "c", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	want = "# Index\n\n## Late\n\n- [m](notes/m.md) — m\n- [z](notes/z.md) — z\n\n## Early\n\n- [b](notes/b.md) — b\n- [C](notes/c.md) — c\n- [d](notes/d.md) — d\n- [Zz](notes/zz.md) — c\n"
	if got := read(t, dir, "kb/index.md"); got != want {
		t.Fatalf("index = %q", got)
	}
}

func TestNewPageUnmatchedGroup(t *testing.T) {
	cfg := "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes, other]\n    kinds:\n      note: {}\n      misc: { folder: other }\nindex:\n  file: index.md\n  check: %s\n  groups:\n    notes: Notes\n"
	files := func(check string) map[string]string {
		return map[string]string{
			"wikai.yaml":            strings.Replace(cfg, "%s", check, 1),
			"kb/_skeletons/note.md": "# {{title}}\n",
			"kb/_skeletons/misc.md": "# {{title}}\n",
			"kb/index.md":           "# Index\n\n## Notes\n\nNo page.\n",
		}
	}
	dir := testrepo.New(t, files("true"))
	b, _ := loadBase(t, dir)
	_, err := NewPage(b, PageOpts{Kind: "misc", Title: "M", Summary: "s", Date: jan2})
	if err == nil || exit.Code(err) != 2 || err.Error() != "no index group for other/: add it to index.groups" {
		t.Fatalf("err = %v", err)
	}
	if exists(dir, "kb/other/m.md") || read(t, dir, "kb/index.md") != "# Index\n\n## Notes\n\nNo page.\n" {
		t.Fatal("something written")
	}
	dir = testrepo.New(t, files("false"))
	b, _ = loadBase(t, dir)
	p, err := NewPage(b, PageOpts{Kind: "misc", Title: "M", Summary: "s", Date: jan2})
	if err != nil || !exists(dir, p) || read(t, dir, "kb/index.md") != "# Index\n\n## Notes\n\nNo page.\n" {
		t.Fatalf("%q %v", p, err)
	}
}

func TestNewPageVaultRefusesWikilinkBreakingName(t *testing.T) {
	b, dir := vaultBase(t)
	for _, title := range []string{"C# basics", "a|b", "a[b", "a]b"} {
		_, err := NewPage(b, PageOpts{Kind: "entity", Title: title, Summary: "s", Date: jan2})
		if exit.Code(err) != 1 || !strings.Contains(err.Error(), "a wikilink name holds none of # | [ ]") {
			t.Errorf("%q: err = %v", title, err)
		}
		if exists(dir, "20-wiki/entites/"+title+".md") {
			t.Errorf("%q written", title)
		}
	}
	if got := read(t, dir, "index.md"); got != vaultIndex {
		t.Fatalf("index changed")
	}
}

func TestNewPageVaultFilenameTitleComparesThePageName(t *testing.T) {
	b, dir := vaultBase(t)
	if _, err := NewPage(b, PageOpts{Kind: "entity", Title: "Talk", Summary: "s", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	b, _ = loadBase(t, dir)
	if p, err := NewPage(b, PageOpts{Kind: "summary", Title: "Talk", Summary: "s", Date: jan2}); err != nil || p != "20-wiki/resumes/Résumé - Talk.md" {
		t.Fatalf("%q %v", p, err)
	}
}

func TestNewPageVaultRefusesAnotherFileWithTheSameName(t *testing.T) {
	b, dir := vaultBase(t)
	if _, err := NewPage(b, PageOpts{Kind: "entity", Title: "Résumé - Q", Summary: "s", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	b, _ = loadBase(t, dir)
	_, err := NewPage(b, PageOpts{Kind: "summary", Title: "Q", Summary: "s", Date: jan2})
	if exit.Code(err) != 1 || !strings.Contains(err.Error(), "20-wiki/entites/Résumé - Q.md") {
		t.Fatalf("err = %v", err)
	}
	if exists(dir, "20-wiki/resumes/Résumé - Q.md") {
		t.Fatal("page written")
	}
	testrepo.Write(t, dir, map[string]string{"_meta/attachments/Notes.md": "x\n"})
	b, _ = loadBase(t, dir)
	if _, err := NewPage(b, PageOpts{Kind: "concept", Title: "notes", Summary: "s", Date: jan2}); exit.Code(err) != 1 {
		t.Fatalf("err = %v", err)
	}
}

func TestNewPageKeepsCRLFIndex(t *testing.T) {
	dir := testrepo.New(t, map[string]string{
		"wikai.yaml":            "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes]\nindex:\n  file: index.md\n  groups:\n    notes: Notes\n",
		"kb/_skeletons/note.md": "# {{title}}\n",
		"kb/index.md":           "# Index\r\n\r\n## Notes\r\n\r\n- [b](notes/b.md) — b\r\n",
		"kb/notes/b.md":         "# b\n",
	})
	b, _ := loadBase(t, dir)
	if _, err := NewPage(b, PageOpts{Kind: "note", Title: "A", Summary: "a", Date: jan2}); err != nil {
		t.Fatal(err)
	}
	want := "# Index\r\n\r\n## Notes\r\n\r\n- [A](notes/a.md) — a\r\n- [b](notes/b.md) — b\r\n"
	if got := read(t, dir, "kb/index.md"); got != want {
		t.Fatalf("index = %q", got)
	}
}

func TestIndexAddWritesABracketedTitleThatParsesBack(t *testing.T) {
	b, _ := homeRepo(t, nil)
	added, err := IndexAdd(b, moveIndex, "knowledge/glossary/foo.md", "Foo [beta]", "x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(added, "- [billing](glossary/billing.md) — billing of a customer\n- [contract](glossary/contract.md) — commercial commitment of a customer\n- [Foo \\[beta\\]](glossary/foo.md) — x\n") {
		t.Fatalf("index = %q", added)
	}
	removed, err := IndexRemove(b, added, "knowledge/glossary/foo.md")
	if err != nil || removed != moveIndex {
		t.Fatalf("index = %q, err = %v", removed, err)
	}
}
