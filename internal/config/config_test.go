package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func mustLoad(t *testing.T, path string) *Config {
	t.Helper()
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return c
}

func TestLoadHome(t *testing.T) {
	c := mustLoad(t, "testdata/home.yaml")
	eq(t, c.Stray, true)
	eq(t, c.BaseBranch, "main")
	eq(t, c.Index.Check, true)
	eq(t, c.Index.OtherLinks, "forbid")
	eq(t, c.Index.Groups.ByFolder, true)
	eq(t, c.Index.Sort, "path")
	eq(t, c.Log.Entry.Separator, ": ")
	eq(t, c.Log.Entry.Labels.Created, "created")
	eq(t, c.Frozen[0], Frozen{Path: "archive", Checked: false})
	eq(t, c.Frozen[1], Frozen{Path: "log", Checked: true})
	eq(t, c.RootPath("glossary"), "knowledge/glossary")
	eq(t, c.Skeletons, "_skeletons")
	eq(t, c.Orphans != nil, true)
	eq(t, c.Manifest == nil, true)
	typ, k, ok := c.KindOf("term")
	eq(t, typ, "term")
	eq(t, k.Folder, "glossary")
	eq(t, k.Name, "{slug}")
	eq(t, ok, true)
	typ, k, _ = c.KindOf("fact")
	eq(t, typ, "decision")
	eq(t, k.Name, "fact-{date:compact}-{slug}")
	eq(t, k.Folder, "decisions")
	_, _, ok = c.KindOf("nothing")
	eq(t, ok, false)
}

func TestLoadDefaults(t *testing.T) {
	c := mustLoad(t, writeFile(t, "minimal.yaml", "version: 1\n"))
	eq(t, c.Root, ".")
	eq(t, c.Locale, "en")
	eq(t, c.Title, "heading")
	eq(t, c.Links, "markdown")
	eq(t, c.Skeletons, "_skeletons")
	eq(t, c.Stray, true)
	eq(t, c.BaseBranch, "main")
	eq(t, c.Index == nil && c.Log == nil && c.Citations == nil && c.Orphans == nil && c.Manifest == nil, true)
	eq(t, c.RootPath("a//b/../c.md"), "a/c.md")
}

func TestLoadBrain(t *testing.T) {
	c := mustLoad(t, "testdata/brain.yaml")
	eq(t, c.Stray, false)
	eq(t, c.Index.Check, false)
	eq(t, c.Index.OtherLinks, "allow")
	eq(t, c.Index.Groups.ByFolder, false)
	eq(t, c.Index.Groups.Headings["20-wiki/resumes"], "Résumés de sources (`20-wiki/resumes/`)")
	eq(t, c.Log.Entry.Separator, " : ")
	eq(t, c.Log.Entry.OneLine, true)
	eq(t, c.Log.Entry.AsLinks, true)
	eq(t, c.Log.Entry.Labels.Created, "créé")
	eq(t, c.RootPath("20-wiki"), "20-wiki")
	eq(t, c.Skeletons, "_meta/skeletons")
	_, k, _ := c.KindOf("entity")
	eq(t, k.Frontmatter["entity_kind"].Values, []string{"person", "tool", "org"})
	eq(t, k.Set, map[string]string{"kind": "entity"})
	eq(t, c.Types["wiki"].Frontmatter["reviewed"], Rule{Required: true, Format: "date", EmptyAllowed: true})
	eq(t, c.Types["wiki"].Frontmatter["sources"].NonEmpty, true)
	eq(t, c.Frozen[0].Except, []string{"10-raw/zotero", "10-raw/AGENTS.md"})
	eq(t, c.Frozen[1], Frozen{Path: "_meta/lint", Checked: true})
	eq(t, c.AppendOnly, []string{"_meta/manifest.jsonl"})
	eq(t, c.TemplateMarkers, []string{"{{", "<%"})
	eq(t, c.Orphans.ExemptTypes, []string{"mission"})
	eq(t, c.Manifest.Statuses, []string{"integrated", "no_material", "error", "superseded"})
	eq(t, c.Protected[len(c.Protected)-1], "wikai.yaml")
	eq(t, c.UncheckedFrontmatter, []string{"00-inbox"})
	eq(t, c.Citations == nil, true)
}

func TestLoadIncludeAndScopeDefaultToTypeFolders(t *testing.T) {
	c := mustLoad(t, "testdata/home.yaml")
	eq(t, c.Index.Include, []string{"decisions", "designs", "domains", "repos", "glossary"})
	eq(t, c.Orphans.Scope, []string{"decisions", "designs", "domains", "repos", "glossary"})
}

func TestLoadRejects(t *testing.T) {
	home, err := os.ReadFile("testdata/home.yaml")
	if err != nil {
		t.Fatal(err)
	}
	base := string(home)
	set := func(edit string) func(string) string {
		return func(b string) string { return replaceTopLevel(b, edit) }
	}
	cases := map[string]struct {
		mutate func(string) string
		want   string
	}{
		"unknown key":      {set("colour: blue\n"), "field colour not found"},
		"unknown nested":   {set("index: {file: i.md, colour: blue}\n"), "field colour not found"},
		"duplicate key":    {func(b string) string { return b + "root: a\n" }, "already defined"},
		"version 2":        {set("version: 2\n"), "expects version 1, found 2"},
		"version missing":  {func(b string) string { return strings.Replace(b, "version: 1\n", "", 1) }, "expects version 1, found none"},
		"version text":     {set("version: x\n"), "expects version 1, found x"},
		"links":            {set("links: html\n"), `links must be one of markdown, wikilink, found "html"`},
		"locale":           {set("locale: de\n"), `locale must be one of en, fr, found "de"`},
		"title":            {set("title: h2\n"), `title must be one of heading, filename, found "h2"`},
		"format":           {set("frontmatter: {x: {format: regex}}\n"), `frontmatter.x.format must be one of`},
		"type format":      {set("types: {a: {frontmatter: {x: {format: regex}}}}\n"), `types.a.frontmatter.x.format must be one of`},
		"log mode":         {set("log: {mode: db, path: l}\n"), `log.mode must be one of dir, file, found "db"`},
		"log path":         {set("log: {mode: dir}\n"), "log.path is required"},
		"other_links":      {set("index: {file: i.md, other_links: x}\n"), `index.other_links must be one of forbid, allow, found "x"`},
		"index file":       {set("index: {check: true}\n"), "index.file is required"},
		"groups":           {set("index: {file: i.md, groups: tree}\n"), `index.groups must be "folder"`},
		"root dotdot":      {set("root: ../x\n"), `root: "../x" must be relative`},
		"root absolute":    {set("root: /x\n"), `root: "/x" must be relative`},
		"ignore dotdot":    {set("ignore: [a/../../b]\n"), `ignore: "a/../../b" must be relative`},
		"folders absolute": {set("types: {a: {folders: [/x]}}\n"), `types.a.folders: "/x" must be relative`},
		"kind folder":      {set("types: {a: {kinds: {k: {folder: ../x}}}}\n"), `types.a.kinds.k.folder: "../x" must be relative`},
		"frozen path":      {set("frozen: [{path: ../x}]\n"), `frozen.path: "../x" must be relative`},
		"frozen except":    {set("frozen: [{path: a, except: [../x]}]\n"), `frozen.except: "../x" must be relative`},
		"protected":        {set("protected: [../x]\n"), `protected: "../x" must be relative`},
		"append_only":      {set("append_only: [/x]\n"), `append_only: "/x" must be relative`},
		"skeletons":        {set("skeletons: ../s\n"), `skeletons: "../s" must be relative`},
		"manifest file":    {set("manifest: {file: ../m}\n"), `manifest.file: "../m" must be relative`},
		"manifest missing": {set("manifest: {pipeline: p}\n"), "manifest.file is required"},
		"kind unique":      {set("types: {a: {kinds: {k: {}}}, b: {kinds: {k: {}}}}\n"), "kind k is declared in types a and b"},
		"kind vs type":     {set("types: {term: {}, a: {kinds: {term: {}}}}\n"), "kind term is declared in types a and term"},
		"frozen no path":   {set("frozen: [{checked: false}]\n"), "frozen.path: path must not be empty"},
		"frozen empty":     {set("frozen: [{path: \"\"}]\n"), "frozen.path: path must not be empty"},
		"except empty":     {set("frozen: [{path: a, except: [\"\"]}]\n"), "frozen.except: path must not be empty"},
		"ignore empty":     {set("ignore: [\"\"]\n"), "ignore: path must not be empty"},
		"protected empty":  {set("protected: [\"\"]\n"), "protected: path must not be empty"},
		"append empty":     {set("append_only: [\"\"]\n"), "append_only: path must not be empty"},
		"unchecked empty":  {set("unchecked_frontmatter: [\"\"]\n"), "unchecked_frontmatter: path must not be empty"},
		"folders empty":    {set("types: {a: {folders: [\"\"]}}\n"), "types.a.folders: path must not be empty"},
		"include empty":    {set("index: {file: i.md, include: [\"\"]}\n"), "index.include: path must not be empty"},
		"scope empty":      {set("orphans: {scope: [\"\"]}\n"), "orphans.scope: path must not be empty"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			file := writeFile(t, "wikai.yaml", tc.mutate(base))
			_, err := Load(file)
			if err == nil {
				t.Fatal("expected an error")
			}
			msg := strings.TrimPrefix(err.Error(), file+": ")
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("error %q does not contain %q", msg, tc.want)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("expected an error")
	}
}

func replaceTopLevel(base, edit string) string {
	key := strings.SplitN(edit, ":", 2)[0]
	if strings.HasPrefix(edit, "version") {
		return strings.Replace(base, "version: 1\n", edit, 1)
	}
	var out []string
	skipping := false
	for _, line := range strings.SplitAfter(base, "\n") {
		top := line != "" && line[0] != ' ' && line[0] != '#' && line[0] != '\n' && line[0] != '-'
		if top {
			skipping = strings.HasPrefix(line, key+":")
		}
		if !skipping {
			out = append(out, line)
		}
	}
	return strings.Join(out, "") + edit
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
