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
	cases := map[string]struct{ edit, want string }{
		"unknown key":       {"colour: blue\n", "colour"},
		"unknown nested":    {"index: {file: i.md, colour: blue}\n", "colour"},
		"duplicate key":     {"root: a\n", "root"},
		"version 2":         {"version: 2\n", "expects version 1, found 2"},
		"version missing":   {"", "expects version 1, found none"},
		"version text":      {"version: x\n", "expects version 1, found x"},
		"links":             {"links: html\n", "links"},
		"locale":            {"locale: de\n", "locale"},
		"title":             {"title: h2\n", "title"},
		"format":            {"frontmatter: {x: {format: regex}}\n", "format"},
		"type format":       {"types: {a: {frontmatter: {x: {format: regex}}}}\n", "format"},
		"log mode":          {"log: {mode: db, path: l}\n", "mode"},
		"log path":          {"log: {mode: dir}\n", "log.path"},
		"other_links":       {"index: {file: i.md, other_links: x}\n", "other_links"},
		"index file":        {"index: {check: true}\n", "index.file"},
		"groups":            {"index: {file: i.md, groups: tree}\n", "groups"},
		"root dotdot":       {"root: ../x\n", "root"},
		"root absolute":     {"root: /x\n", "root"},
		"ignore dotdot":     {"ignore: [a/../../b]\n", "ignore"},
		"folders absolute":  {"types: {a: {folders: [/x]}}\n", "folders"},
		"kind folder":       {"types: {a: {kinds: {k: {folder: ../x}}}}\n", "folder"},
		"frozen path":       {"frozen: [{path: ../x}]\n", "frozen"},
		"protected":         {"protected: [../x]\n", "protected"},
		"append_only":       {"append_only: [/x]\n", "append_only"},
		"skeletons":         {"skeletons: ../s\n", "skeletons"},
		"manifest file":     {"manifest: {file: ../m}\n", "manifest.file"},
		"manifest required": {"manifest: {pipeline: p}\n", "manifest.file"},
		"kind unique":       {"types: {a: {kinds: {k: {}}}, b: {kinds: {k: {}}}}\n", "kind k"},
		"kind vs type":      {"types: {term: {}, a: {kinds: {term: {}}}}\n", "kind term"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			src := replaceTopLevel(base, tc.edit)
			if name == "version missing" {
				src = strings.Replace(base, "version: 1\n", "", 1)
			}
			if name == "duplicate key" {
				src = base + tc.edit
			}
			_, err := Load(writeFile(t, "wikai.yaml", src))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
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
