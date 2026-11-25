# wikai core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `wikai`, one Go binary that checks, writes and guards any LLM wiki described by a `wikai.yaml`, and release it through Homebrew.

**Architecture:** A config package turns `wikai.yaml` into a validated struct. A wiki package loads a base (pages, frontmatter, links) once. Each check is a function of the loaded base that returns findings. The write, manifest and guard packages reuse the same loaded base. cobra commands only parse flags, call a package, print, and map errors to exit codes.

**Tech Stack:** Go 1.27.4, cobra v1.10.2, go.yaml.in/yaml/v4 v4.0.0, golang.org/x/text v0.44.0, goreleaser v2, GitHub Actions, govulncheck v1.10.0.

**Spec:** `docs/superpowers/specs/2026-11-25-wikai-core-design.md`. The checklist `docs/superpowers/specs/2026-11-25-wikai-core-validation.md` wins over the spec. The checklist of this plan, `docs/superpowers/plans/2026-11-25-wikai-core-validation.md` (approved 2026-11-25), wins over this plan and over the spec where they differ. Executors read all three.

**Reference sources to port (read-only, never copy a real name out of them):**

- `~/Development/Pillow/am-home/knowledge/_ops/check.py`, `new.py`, `tests/`, `fixtures/` (29 fixtures)
- `~/Development/Digitaleo/dev-home/knowledge/_ops/` — diffed against am-home on 2026-11-25: `check.py` differs only by the absent code citations, `origin/master` tried before `origin/main`, and the `link leaves …` wording; `new.py` differs only by the French texts. Its 26 fixtures are the am-home ones without the three `code-citation-*`. Port from am-home only.
- `~/Brain/_meta/scripts/health_check.py`, `manifest.py`, `protect_raw.py`, `guard.sh`
- `~/Development/Pillow/den/.goreleaser.yaml`, `.github/workflows/`

## Global Constraints

- Module `github.com/PillowPillow/wikai`. Repository `git@github.com:PillowPillow/wikai.git`.
- Versions, checked at the source on 2026-11-25 (go.dev, proxy.golang.org, GitHub releases, vuln.go.dev):

  | Item | Version |
  |---|---|
  | Go (`go` line of `go.mod`) | `1.27.4` |
  | `github.com/spf13/cobra` | `v1.10.2` |
  | `github.com/spf13/pflag` (indirect, raised by hand) | `v1.1.0` |
  | `github.com/inconshreveable/mousetrap` (indirect) | `v1.1.0` |
  | `go.yaml.in/yaml/v4` | `v4.0.0` |
  | `golang.org/x/text` | `v0.44.0` |
  | `golang.org/x/vuln/cmd/govulncheck` (run with `go run`, not in `go.mod`) | `v1.10.0` |
  | `actions/checkout`, `actions/setup-go`, `goreleaser/goreleaser-action` | `v7`, `v7`, `v7` |
  | goreleaser | `~> v2` (latest: v2.20.3) |

- **Deviation from the spec, approved by the human:** the spec names `gopkg.in/yaml.v3`, whose upstream repository is archived. The plan uses `go.yaml.in/yaml/v4`, the maintained fork in its latest major version. Read `docs/v3-to-v4-migration.md` of `github.com/yaml/go-yaml` (tag `v4.0.0`) before Task 2: load with `yaml.Load(data, &out, yaml.WithKnownFields(), yaml.WithUniqueKeys())`; the signature of a custom unmarshaler differs from v3.
- No other dependency. The file lock uses `syscall.Flock` from the standard library.
- Zero known vulnerability: `go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...` runs in CI on every push and before a release. One finding fails the build.
- git is the only external program. Every git call goes through `internal/gitx`, with `-z` output where git offers it and `-c core.quotePath=false`.
- No call to a model or to the network.
- Exit codes of every command: 0 success, 1 refused or findings, 2 usage or environment error.
- Output is deterministic: findings sorted by path, then by check.
- CLI messages are English. `locale` only sets text that `wikai` writes into pages.
- An unknown key in `wikai.yaml` is an error.
- Tests and fixtures never contain a real client, person, repo or host name. Use `example-backend`, `acme`, `git@example.com:acme/example-backend.git`.
- Code comments: none by default. One line only for a *why* the code cannot show.
- Path rule (approved by the human; the spec text and its example disagree): `root` and `protected` are relative to the boundary. Every other path in `wikai.yaml` is relative to `root`. Paths in findings, in the manifest, and in command output are relative to the boundary. A path argument on the command line is relative to the current folder and is converted to a boundary-relative path.

## Review Focus

1. A page with a BOM or CRLF line ends: the frontmatter is still found and parsed. (Task 3)
2. A page that is not UTF-8, or whose frontmatter is valid YAML but not a mapping: one `frontmatter` finding, no crash. (Tasks 3, 4)
3. A file name with spaces, accents, or NFD composition (macOS): git output is read without quoting, and a wikilink written in NFC resolves to the NFD file. (Tasks 4, 6)
4. `guard hook` with an invalid `wikai.yaml`, with malformed stdin, or with a target reached through a symlink: it refuses (exit 2). With no `wikai.yaml` at all it allows (exit 0). (Task 14)
5. `new page` or `move` with a title or target that leaves its folder (`a/b`, `../x`), or run from a subfolder of the base: refused with nothing written, and path arguments resolve from the current folder. (Tasks 10, 12)

## File Structure

```
cmd/wikai/main.go             os.Exit(cli.Execute(os.Args[1:], real Env))
internal/exit/exit.go         Refused / Usage errors, Code(err)
internal/cli/root.go          root command, Env, --config/--dir, --version, loadBase
internal/cli/{check,new,move,manifest,guard,init}.go   one file per command group
internal/config/config.go     types, Load, defaults, validation
internal/config/locate.go     Find (walk up), Locate (--config/--dir)
internal/wiki/frontmatter.go  split and parse frontmatter
internal/wiki/text.go         code fences and spans, headings, anchors, block ids, title
internal/wiki/links.go        markdown links, wikilinks, RewriteLinks
internal/wiki/citations.go    (source: X) parser
internal/wiki/base.go         Base, Page, Doc, Load, Resolve, Rel
internal/check/check.go       Finding sort, registry, Run
internal/check/{stray,frontmatter,links,index,duplicates,orphans,sources,templates,frozen}.go
internal/check/testdata/      home.yaml, fixtures/<name>/{tree/,base/,expected.json,wikai.yaml?}
internal/write/{slug,index,page,log,move}.go
internal/manifest/manifest.go
internal/guard/{rules,hook,commit}.go
internal/gitx/gitx.go
internal/preset/preset.go     embed.FS + Write
internal/preset/{home-repo,obsidian-vault}/
.github/workflows/{gates,ci,release}.yml
.goreleaser.yaml
acceptance/run.sh               (the three configs stay outside the repository)
```

Blocking edges: Tasks 1 → 2 → 3 → 4 are sequential. Tasks 5–9 (checks) need 4 and are sequential only because they share the fixture harness of Task 5. Tasks 10–12 (write) need 4. Task 13 needs 2. Task 14 needs 2 and the gitx of Task 4. Task 15 needs 10. Task 16 needs 1. Task 17 needs 5–9.

---

### Task 1: Module, entry point, exit codes, CI gate

**Files:**
- Create: `go.mod`, `cmd/wikai/main.go`, `internal/exit/exit.go`, `internal/cli/root.go`, `.gitignore`, `.github/workflows/gates.yml`, `.github/workflows/ci.yml`
- Test: `internal/exit/exit_test.go`, `internal/cli/root_test.go`

**Interfaces:**
- Produces:
  ```go
  // internal/exit
  func Refused(format string, a ...any) error   // exit 1
  func Usage(format string, a ...any) error     // exit 2
  func Code(err error) int                      // nil→0, Refused→1, anything else→2

  // internal/cli
  var Version = "dev"
  type Env struct {
      Dir    string                // current folder
      Stdin  io.Reader
      Stdout, Stderr io.Writer
      Getenv func(string) string
      Now    func() time.Time
  }
  func Execute(args []string, env Env) int
  ```

- [ ] **Step 1: Write the failing tests**

```go
// internal/exit/exit_test.go
func TestCode(t *testing.T) {
    if Code(nil) != 0 || Code(Refused("x")) != 1 || Code(Usage("x")) != 2 ||
        Code(errors.New("io")) != 2 || Code(fmt.Errorf("wrap: %w", Refused("x"))) != 1 {
        t.Fatal("wrong exit code mapping")
    }
}

// internal/cli/root_test.go — helper used by every later CLI test
func run(t *testing.T, dir string, args ...string) (code int, out, errOut string)

func TestVersion(t *testing.T)        { /* run("--version") → 0, out == "dev\n" */ }
func TestUnknownCommand(t *testing.T) { /* run("nope") → 2, errOut starts with "error: " */ }
```

- [ ] **Step 2: Run** `go test ./...` — Expected: FAIL, packages do not compile.

- [ ] **Step 3: Implement**

`go.mod`: `module github.com/PillowPillow/wikai`, `go 1.27.4`. Run `go get github.com/spf13/cobra@v1.10.2 github.com/spf13/pflag@v1.1.0`. 
Root command: `SilenceUsage` and `SilenceErrors` on; `SetVersionTemplate("{{.Version}}\n")`; persistent flags `--config`, `--dir`. `Execute` prints `error: <msg>` (or `refused: <msg>` for a Refused error) on `env.Stderr` and returns `exit.Code(err)`. A cobra flag or argument error is a usage error.

`.gitignore`: `/wikai`, `/dist/`.

`gates.yml` (`on: workflow_call`), one job on `ubuntu-latest`: `actions/checkout@v7`; `actions/setup-go@v7` with `go-version-file: go.mod`; then `go vet ./...`; `test -z "$(gofmt -l .)"`; `go test -count=1 ./...`; `go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...`.
`ci.yml`: `on: [push, pull_request]`, one job `uses: ./.github/workflows/gates.yml`.

- [ ] **Step 4: Run** `go vet ./... && test -z "$(gofmt -l .)" && go test -count=1 ./... && go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...` — Expected: PASS, and `No vulnerabilities found.`

- [ ] **Step 5: Commit** — `git commit -m "feat: module, entry point, exit codes and CI gate"`

---

### Task 2: `wikai.yaml` — load, defaults, validation, location

**Files:**
- Create: `internal/config/config.go`, `internal/config/locate.go`
- Test: `internal/config/config_test.go`, `internal/config/locate_test.go`, `internal/config/testdata/{home.yaml,brain.yaml}`

**Interfaces:**
- Produces:
  ```go
  type Rule struct {
      Required     bool     `yaml:"required"`
      Format       string   `yaml:"format"`        // "" (text) | date | list
      Values       []string `yaml:"values"`
      NonEmpty     bool     `yaml:"non_empty"`
      EmptyAllowed bool     `yaml:"empty_allowed"`
  }
  type Kind struct {
      Folder, Name, Skeleton string
      Set         map[string]string
      Frontmatter map[string]Rule
  }
  type Type struct {
      Folders     []string
      Frontmatter map[string]Rule
      Kinds       map[string]Kind
  }
  type Groups struct { ByFolder bool; Headings map[string]string } // yaml: "folder" or a map
  type Index struct {
      File string; Check bool; Include []string
      OtherLinks string // forbid | allow
      Groups Groups; Sort string
  }
  type Entry struct {
      Labels struct{ Created, Modified, Note string }
      Separator string; OneLine, AsLinks bool
  }
  type Log struct { Mode, Path string; Operations []string; Entry Entry } // mode: dir | file
  type Frozen struct { Path string; Checked bool; Except []string }
  type Citations struct { Frontmatter string; Code bool }
  type Orphans struct { Scope, IgnoreLinksFrom, IgnoreLinksFromTypes, ExemptTypes []string }
  type Manifest struct { File, Pipeline string; Statuses []string }
  type Config struct {
      Version int; Root, Locale, Title, Links string
      Ignore, UncheckedFrontmatter []string
      Frontmatter map[string]Rule
      Types map[string]Type
      Stray bool; Skeletons string
      Index *Index; Log *Log
      Frozen []Frozen; AppendOnly []string; BaseBranch string
      Citations *Citations; Orphans *Orphans
      TemplateMarkers []string
      Manifest *Manifest; Protected []string
  }
  func Load(path string) (*Config, error)
  func (c *Config) RootPath(p string) string           // root-relative → boundary-relative, slash, cleaned
  func (c *Config) KindOf(name string) (typeName string, k Kind, ok bool)
  func Find(startDir string) (file string, err error)  // walks up to a wikai.yaml
  func Locate(cwd, configFlag, dirFlag string) (cfg *Config, boundary string, err error)
  ```
- A nil `Index`, `Log`, `Citations`, `Orphans` or `Manifest`, and an empty `Frozen`, `AppendOnly`, `TemplateMarkers` or `Protected`, mean the mechanism is off.

- [ ] **Step 1: Write the failing tests**

`testdata/home.yaml` is the home example of the spec, plus `domain: { folders: [domains] }` and `ignore: [README.md, _ops]`. `testdata/brain.yaml` is the home example with the Brain differences of the spec applied, the `# …` type comment removed, and `skeletons: _meta/skeletons`.

```go
func TestLoadHome(t *testing.T) {
    c := mustLoad(t, "testdata/home.yaml")
    // defaults
    eq(t, c.Stray, true); eq(t, c.BaseBranch, "main"); eq(t, c.Index.Check, true)
    eq(t, c.Index.OtherLinks, "forbid"); eq(t, c.Index.Groups.ByFolder, true)
    eq(t, c.Log.Entry.Separator, ": ")
    eq(t, c.Log.Entry.Labels.Created, "created")
    eq(t, c.Frozen[0], Frozen{Path: "archive", Checked: false})
    eq(t, c.Frozen[1], Frozen{Path: "log", Checked: true})
    eq(t, c.RootPath("glossary"), "knowledge/glossary")
    // a type without kinds has one kind named after it
    typ, k, ok := c.KindOf("term"); eq(t, typ, "term"); eq(t, k.Folder, "glossary"); eq(t, k.Name, "{slug}"); eq(t, ok, true)
    typ, k, _ = c.KindOf("fact"); eq(t, typ, "decision"); eq(t, k.Name, "fact-{date:compact}-{slug}"); eq(t, k.Folder, "decisions")
}
func TestLoadBrain(t *testing.T) {
    c := mustLoad(t, "testdata/brain.yaml")
    eq(t, c.Stray, false); eq(t, c.Index.Check, false); eq(t, c.Index.OtherLinks, "allow")
    eq(t, c.Index.Groups.Headings["20-wiki/resumes"], "Résumés de sources (`20-wiki/resumes/`)")
    eq(t, c.Log.Entry.Separator, " : "); eq(t, c.Log.Entry.OneLine, true)
    eq(t, c.RootPath("20-wiki"), "20-wiki")
    _, k, _ := c.KindOf("entity")
    eq(t, k.Frontmatter["entity_kind"].Values, []string{"person", "tool", "org"})
}
func TestLoadRejects(t *testing.T) { // each case: a one-key change to home.yaml, and a substring of the error
    cases := map[string]string{
        "colour: blue":                        "colour",          // unknown key
        "version: 2":                          "expects version 1",
        "(the version line removed)":          "expects version 1",   // the key is required
        "links: html":                         "links",
        "locale: de":                          "locale",
        "title: h2":                           "title",
        "frontmatter: {x: {format: regex}}":   "format",
        "log: {mode: db, path: l}":            "mode",
        "index: {file: i.md, other_links: x}": "other_links",
        "root: ../x":                          "root",            // any path that is absolute or holds ".."
        "types: {a: {kinds: {k: {}}}, b: {kinds: {k: {}}}}": "kind k", // a kind name is unique across types
    }
}
func TestFind(t *testing.T)   { /* wikai.yaml in tmp/a; Find(tmp/a/b/c) → tmp/a/wikai.yaml; Find(other tmp) → error containing "no wikai.yaml" */ }
func TestLocate(t *testing.T) { /* (cwd, "", "") → boundary = folder of the found file
                                   (cwd, f, "") → boundary = cwd; (cwd, f, d) → boundary = d
                                   (cwd, "", d) → Usage error: --dir needs --config */ }
```

- [ ] **Step 2: Run** `go test ./internal/config/` — Expected: FAIL, undefined.

- [ ] **Step 3: Implement.** Decode with `yaml.Load` and `yaml.WithKnownFields()`. The `version` key is required: a missing or unknown version gives `<file>: wikai expects version 1, found <value or "none">`. No migration command is written in this project. `Groups` and the `Checked`/`Check`/`Stray` defaults of `true` need custom unmarshalling (decode into pointer shadows, then fill). Defaults: `root "."`, `locale en`, `title heading`, `links markdown`, `skeletons "_skeletons"`, `index.sort path`. Every error is `exit.Usage`.

- [ ] **Step 4: Run** `go test ./internal/config/` — Expected: PASS.

- [ ] **Step 5: Commit** — `git commit -m "feat: wikai.yaml loading, defaults and validation"`

---

### Task 3: Text primitives — frontmatter, code, links, anchors, citations

**Files:**
- Create: `internal/wiki/frontmatter.go`, `text.go`, `links.go`, `citations.go`
- Test: one `_test.go` per file

**Interfaces:**
- Produces:
  ```go
  // frontmatter.go
  // meta == nil && err == nil: no frontmatter. Scalars are kept as their source text
  // (string, null → ""); a sequence of scalars is []string; any other node is its decoded value.
  func ParseFrontmatter(text string) (meta map[string]any, raw, body string, err error)

  // text.go
  type Line struct { Number int; Text string }      // code spans blanked with spaces
  func OutsideCode(text string) []Line              // skips fenced blocks
  func HeadingTitle(body string) (string, bool)     // first "# " line outside code
  func Anchors(text string) map[string]bool
  func Headings(text string) []string               // heading texts outside code, in order
  func HasBlockID(text, id string) bool             // "^id" at the end of a line

  // links.go
  type Link struct {
      Line int
      Target string   // markdown: decoded path without ?query; wikilink: name as written, trimmed
      Anchor string; HasAnchor bool
      External bool   // markdown only: empty target or a URL scheme
      Unparsed bool   // markdown only: a "](" that no link matches
  }
  func Links(style, text string) []Link             // style: markdown | wikilink
  // fn receives each parsed link and returns the new target; ok=false keeps the link as written.
  func RewriteLinks(style, text string, fn func(Link) (target string, ok bool)) string

  // citations.go
  type Citation struct { Line int; Key string; Unparsed bool }
  func Citations(text string) []Citation
  ```

Rules the tests do not fully determine (ported from `check.py` and `health_check.py`):

- Frontmatter: strip leading BOMs; the first line is `---`; the block ends at the next line that is exactly `---`; line ends may be LF or CRLF. The YAML root must be a mapping. Decode through `yaml.Node`, never into `any` directly: yaml turns `2026-01-01` into a timestamp, and the date check needs the source text.
- Fence: `^\s*(`{3,}|~{3,})`; it closes on a line whose fence uses the same character and is at least as long.
- Code span: a run of N backticks up to the next run of N backticks on the same line. Go's `regexp` has no backreference: write a scanner.
- Markdown link: `!?\[[^\]]*\]\(\s*(<[^>]*>|[^)\s]*)(?:\s+(?:"[^"]*"|'[^']*'))?\s*\)`. External when the target is empty or matches `^[A-Za-z][A-Za-z0-9+.-]*:`. Path = text before `#`, `?query` dropped, percent-decoded; anchor percent-decoded.
- Wikilink: `\[\[([^\[\]|#]*)(?:#([^\[\]|]*))?(?:\|[^\[\]]*)?\]\]`. Target trimmed, one trailing `\` removed (table escape). Wikilinks are also read in the frontmatter text; code is ignored in the body only.
- Heading: `^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$`. Anchor slug: lowercase; keep space, `-`, `_`, and every Unicode letter or number; then space → `-`. A repeated slug takes `-1`, `-2`.
- Citation: `(source: X)` outside code. `X` is a markdown link (the key is its target) or text with at most one level of nested parentheses. A `(source:` with no parsed citation is `Unparsed`. Go's `regexp` has no lookahead: write a scanner.

- [ ] **Step 1: Write the failing tests**

```go
func TestParseFrontmatter(t *testing.T) {
    meta, _, body, err := ParseFrontmatter("---\ntype: term\ncreated: 2026-01-01\naliases: [subscription, 'offer']\nsources:\n  - a.md\norigin:\nempty: []\n---\n# T\n")
    // err == nil; body == "# T\n"
    // meta == {"type":"term","created":"2026-01-01","aliases":[]string{"subscription","offer"},
    //          "sources":[]string{"a.md"},"origin":"","empty":[]string{}}
}
func TestFrontmatterEdges(t *testing.T) {
    // "# T\n"                              → meta nil, err nil
    // "﻿---\r\ntype: term\r\n---\r\n# T\r\n" → meta["type"] == "term"
    // "---\ntype: [a\n---\n"               → err != nil
    // "---\n- a\n- b\n---\n"               → err contains "not a mapping"
    // "---\ntype: term\n"  (never closed)  → meta nil, err nil
}
func TestAnchors(t *testing.T) {
    // "# Définition\n## Two words\n## Two words\n```\n# not a heading\n```\n"
    // → {"définition", "two-words", "two-words-1"}
}
func TestMarkdownLinks(t *testing.T) {
    text := "See [a](../x/a%20b.md#Sec) and ![i](<img one.png> \"t\") and [u](https://example.com) and [s](#top).\n" +
        "`[c](code.md)` text\n```\n[f](fenced.md)\n```\nbroken [x](a b c\n"
    // → {1,"../x/a b.md","Sec"}, {1,"img one.png"}, {1,External}, {1,"",Anchor "top"} (not External: self link),
    //   {6,Unparsed}
}
func TestWikilinks(t *testing.T) {
    text := "---\nsources: [\"[[Raw note]]\"]\n---\n[[Page]] [[Dir/Page#^b1|shown]] ![[img.png]] `[[code]]` | [[T\\|x]] |\n"
    // → targets "Raw note", "Page", "Dir/Page" (Anchor "^b1"), "img.png", "T"
}
func TestRewriteLinks(t *testing.T) {
    // markdown: "[a](old.md#s) `[a](old.md)`" with fn old.md→new/n.md → "[a](new/n.md#s) `[a](old.md)`"
    // wikilink: "[[Old#^b|t]] [[Other]]" with fn Old→New → "[[New#^b|t]] [[Other]]"
}
func TestCitations(t *testing.T) {
    text := "A (source: repo@1a2b3c4:a.md#L1-L2). B (source: [doc](../x.md)). C (source: f(x) note). D (source: \n`(source: code)`\n"
    // → {1,"repo@1a2b3c4:a.md#L1-L2"}, {1,"../x.md"}, {1,"f(x) note"}, {1,Unparsed}
}
func TestHasBlockID(t *testing.T) { /* "line ^abc-1  \n" has "abc-1"; "^abc-1 mid line\n" has not */ }
```

- [ ] **Step 2: Run** `go test ./internal/wiki/` — Expected: FAIL, undefined.
- [ ] **Step 3: Implement** the four files.
- [ ] **Step 4: Run** `go test ./internal/wiki/` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: frontmatter, link, anchor and citation parsing"`

---

### Task 4: git calls and base loading

**Files:**
- Create: `internal/gitx/gitx.go`, `internal/wiki/base.go`
- Test: `internal/gitx/gitx_test.go`, `internal/wiki/base_test.go`, `internal/testrepo/testrepo.go` (test helper package)

**Interfaces:**
- Consumes: `config.Config`, Task 3 functions.
- Produces:
  ```go
  // internal/testrepo — used by every later test that needs a repository
  func New(t *testing.T, files map[string]string) string  // temp dir, git init -b main, writes files, commits them
  func Git(t *testing.T, dir string, args ...string) string // runs git with a fixed test identity, signing off
  func Write(t *testing.T, dir string, files map[string]string)

  // internal/gitx
  type Change struct { Status, Path string }
  func IsRepo(dir string) bool
  func HasCommit(dir string) bool
  func LsFiles(dir string) ([]string, error)                 // tracked or staged, relative to dir
  func LsTree(dir, ref string, paths ...string) ([]string, error)
  func Show(dir, ref, path string) ([]byte, error)           // ref "" reads the index
  func MergeBase(dir, a, b string) (string, error)
  func Diff(dir string, args ...string) ([]Change, error)    // --name-status --no-renames -z
  func Mv(dir, from, to string) error
  // ref "" means: skip the frozen check.
  func BaseRef(dir, baseBranch, flag string) (ref string, warnings []string, err error)

  // internal/wiki
  type Finding struct {
      Check   string `json:"check"`
      Path    string `json:"path"`
      Message string `json:"message"`
  }
  type Page struct {
      Path, Text, Body, RawMeta string
      Meta    map[string]any   // nil: no frontmatter, or MetaErr != nil
      MetaErr error
      Title   string           // "" when title: heading and the page has no "# " line
      Type    string           // Meta["type"] when it is text
      Folder  string           // root-relative type folder that holds the page, "" if none
      Frozen, Unchecked bool   // in a frozen path / in an unchecked_frontmatter folder
  }
  type Doc struct { Path, Text string; OutsideOnly bool }  // a non-page file whose links are read
  type Base struct {
      Boundary string; Cfg *config.Config
      Git   bool
      Files map[string]bool     // link targets, boundary-relative
      Pages []*Page             // sorted by Path
      Docs  []*Doc
      Stray []string            // markdown files in no type folder, when stray is on
      LoadFindings []Finding
  }
  type Resolution int
  const ( Resolved Resolution = iota; Skipped; Outside; NotFound )
  func Load(boundary string, cfg *config.Config) (*Base, error)
  func (b *Base) Page(path string) *Page
  func (b *Base) IndexPath() string   // boundary-relative, "" when the index is off
  func (b *Base) LogPath() string
  func (b *Base) Resolve(from string, l Link) (target string, r Resolution)
  func (b *Base) Rel(cwd, arg string) (string, error) // Usage error when arg leaves the boundary
  ```

Loading rules:

- Markdown files are walked on disk under `root`, minus `.git` and `ignore`. `Path` is NFC.
- Not a page, not a doc: files under `skeletons`; log entries when `log.mode` is `dir`.
- Doc with `OutsideOnly`: files of a frozen path with `checked: false`.
- Doc: the index file and the log file (`file` mode) when they have no frontmatter. With a frontmatter they are pages.
- With `stray: true`, a markdown file that is in no type folder and no frozen path, and is none of the above, goes to `Stray` and is not a page. (The ported fixture `stray-markdown` expects one finding per such file and no other.) With `stray: false` it is a page.
- A file that is not UTF-8: `Finding{"frontmatter", path, "not UTF-8"}` in `LoadFindings`, and the file is dropped.
- `Title`: `heading` → `HeadingTitle(Body)`; `filename` → the file stem.
- `Files`: `markdown` style → `gitx.LsFiles` when the boundary is in a repository, keeping only files that exist on disk; a disk walk otherwise. `wikilink` style → always a disk walk minus `.git`.
- `Resolve`, markdown: `Skipped` for an external link; a target without path is the page itself; resolve from the folder of `from`; `Outside` when the cleaned path leaves the boundary; `NotFound` unless the exact path is in `Files`.
- `Resolve`, wikilink: empty target → the page itself. Compare in NFC and lower case, with a trailing `.md` removed. A file matches by its whole path, by its path without `.md`, by its stem (markdown file) or its file name (other file), else by path suffix `"/" + target`. The shortest path wins.
- `BaseRef`: no repository → `""` and the warning `not a git repository, frozen check skipped`; no commit → `""`; `flag` set → `flag`; else the merge-base of `HEAD` and `origin/<baseBranch>`; no remote → `"HEAD"` and the warning `no remote, base is HEAD`; a remote without merge-base → error `no merge-base with origin/<baseBranch>; fetch origin or pass --base`.

- [ ] **Step 1: Write the failing tests**

```go
// gitx_test.go
func TestBaseRef(t *testing.T) {
    // plain temp dir            → "", ["not a git repository, frozen check skipped"], nil
    // repo, no commit           → "", nil, nil
    // repo with a commit        → "HEAD", ["no remote, base is HEAD"], nil
    // same, flag "abc"          → "abc", nil, nil
    // remote "origin" added, never fetched → err contains "no merge-base with origin/main"
    // clone of a bare remote, one local commit on top → the remote commit's hash
}
func TestLsFilesOddNames(t *testing.T) {
    // commit "dir/é à.md" and stage "new file.md" → LsFiles returns both, unquoted
}
func TestDiffAndShow(t *testing.T) {
    // commit a.md; then modify a.md, add b.md, delete nothing; git add -A
    // Diff(dir, "--cached") == [{M a.md} {A b.md}]; Show(dir, "HEAD", "a.md") is the old content; Show(dir, "", "a.md") the new
}

// base_test.go — a home tree built with testrepo.New and config testdata/home.yaml copied from Task 2
func TestLoadHome(t *testing.T) {
    files := map[string]string{
        "knowledge/index.md":                "# Index\n",
        "knowledge/README.md":               "readme",
        "knowledge/_skeletons/term.md":      "---\ntype: term\n---\n# {{title}}\n",
        "knowledge/glossary/contract.md":    "---\ntype: term\n---\n# Contract\n",
        "knowledge/archive/old.md":          "# Old\n",
        "knowledge/log/2026-01-01-init-x.md": "## [2026-01-01] init | x\n",
        "knowledge/notes.md":                "stray",
        "knowledge/glossary/latin1.md":      "caf\xe9",
    }
    // Pages: exactly knowledge/glossary/contract.md, Title "Contract", Type "term", Folder "glossary"
    // Docs: knowledge/archive/old.md (OutsideOnly), knowledge/index.md
    // Stray: ["knowledge/notes.md"]
    // LoadFindings: [{"frontmatter","knowledge/glossary/latin1.md","not UTF-8"}]
    // IndexPath() == "knowledge/index.md"
}
func TestResolveMarkdown(t *testing.T) {
    // from knowledge/glossary/contract.md:
    // "../index.md" → knowledge/index.md, Resolved;  "../../../x.md" → Outside
    // "Contract.md" → NotFound (wrong case, even on a case-insensitive disk);  "" + anchor → the page itself
    // a file on disk that git does not track → NotFound
}
func TestResolveWikilink(t *testing.T) {
    // vault with "20-wiki/concepts/Café.md" written to disk in NFD, "20-wiki/a/Note.md", "x/20-wiki/a/Note.md", "img/pic.png"
    // "café" (NFC) → 20-wiki/concepts/Café.md;  "Note" → 20-wiki/a/Note.md (shortest)
    // "a/Note" → 20-wiki/a/Note.md;  "pic.png" → img/pic.png;  "Missing" → NotFound
}
func TestRel(t *testing.T) {
    // cwd = boundary/knowledge/glossary: Rel(cwd, "contract.md") == "knowledge/glossary/contract.md"
    // Rel(cwd, "../../../x.md") → Usage error
}
```

- [ ] **Step 2: Run** `go test ./internal/gitx/ ./internal/wiki/` — Expected: FAIL, undefined.
- [ ] **Step 3: Implement** `testrepo`, `gitx`, `base.go`.
- [ ] **Step 4: Run** `go test ./internal/...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: git calls and base loading"`

---

### Task 5: `wikai check`, fixture harness, `stray` and `frontmatter`

**Files:**
- Create: `internal/check/check.go`, `stray.go`, `frontmatter.go`, `internal/cli/check.go`
- Create: `internal/check/testdata/home.yaml`, `internal/check/testdata/fixtures/…`
- Test: `internal/check/fixtures_test.go`, `internal/cli/check_test.go`

**Interfaces:**
- Consumes: `wiki.Load`, `wiki.Base`, `gitx.BaseRef`, `cli.Execute`.
- Produces:
  ```go
  // internal/check
  type Check struct { Name string; Run func(*wiki.Base) []wiki.Finding }
  func All() []Check   // later tasks append to this list
  // LoadFindings + every check + the frozen check (Task 9), sorted by Path then Check.
  func Run(b *wiki.Base, baseFlag string) (findings []wiki.Finding, warnings []string, err error)
  func Stray(b *wiki.Base) []wiki.Finding
  func Frontmatter(b *wiki.Base) []wiki.Finding
  ```
- CLI: `wikai check [--base <ref>] [--json]`. Warnings go to stderr as `warning: <text>`. Text output: `<check>: <path>: <message>` per finding, then `FAIL: <n> finding(s)` (exit 1) or `OK: <n> pages` (exit 0), `n` = `len(b.Pages)`. `--json`: the finding array, indented by two spaces, `[]` when empty.

`internal/check/testdata/home.yaml` is a copy of `internal/config/testdata/home.yaml` (Task 2).

Fixture format: `fixtures/<name>/tree/` is the working tree, `base/` (optional) is the committed base state, `expected.json` the findings, `wikai.yaml` (optional) replaces `testdata/home.yaml`.

Harness, per fixture: `testrepo.New` with the files of `base/` (an empty commit when there is none); delete every working file; copy `tree/` and the config as `wikai.yaml`; `git add -A`; run `cli.Execute(["check","--json"])` in that folder. Match as the Python test does: each expected entry matches exactly one finding with the same `check` and `path` and a `message` that contains the expected text; no finding remains.

Porting rule for an am-home fixture: `knowledge/` → `tree/knowledge/`; `am-backend` → `example-backend`; `git@github.com:PillowPillow/am-backend.git` → `git@example.com:acme/example-backend.git`; any other real name → a fictitious one; French text → English; check names `archive` and `log` → `frozen`; messages `archive file …` and `log entry …` → `frozen file …`; `link leaves am-home` → `link leaves the base`.

Messages:

| Check | Message |
|---|---|
| `stray` | `markdown file outside the page folders` |
| `frontmatter` | `no frontmatter` · `invalid YAML: <error>` · `missing key: <key>` · `<key> is not YYYY-MM-DD` · `<key> is not a list` · `<key> is empty` · `<key> must be one of <a, b>, found "<v>"` · `unknown type: "<v>"` · `type must be <t1 or t2> in <folder>/, found "<v>"` |

Frontmatter rules: skip `Unchecked` pages. The rule set of a page = top-level rules, then the rules of its type, then the rules of each kind of that type whose `set` values all equal the page's values. `required`: the key is present. `format: date`: the text matches `^\d{4}-\d{2}-\d{2}$` and is a real date; an empty value passes only with `empty_allowed`. `format: list`: the value is a list. `values`: same empty rule as `date`. `non_empty`: `""` and an empty list are findings. Type: an undeclared type is `unknown type`; a page that sits in a type folder whose types do not include the page's type, while its type has `folders`, is `type must be …` (the types that own that folder, sorted). A page with an `unknown type` or a `type must be …` finding receives the top-level rules only: the ported fixture `frontmatter-wrong-type` (a `type: repo` page in `glossary/`, without `origin`) expects that one finding.

- [ ] **Step 1: Port and write the fixtures, and write the failing tests**

Port from am-home: `valid`, `stray-markdown`, `frontmatter-invalid-date`, `frontmatter-missing-key`, `frontmatter-none`, `frontmatter-wrong-type`, `repo-without-aliases`, `repo-without-origin`, `term-without-aliases`.

New fixtures, each with its own `wikai.yaml` (wikilink style, `root: .`, `stray: false`, the `wiki` type of the Brain example with its three kinds, `unchecked_frontmatter: [00-inbox]`):

| Fixture | Tree | Expected |
|---|---|---|
| `frontmatter-invalid-yaml` | home tree; `contract.md` frontmatter holds `aliases: [a` | `frontmatter` · `knowledge/glossary/contract.md` · `invalid YAML` |
| `vault-conditional-key` | an entity page without `entity_kind`; a concept page without it | one finding on the entity page: `missing key: entity_kind` |
| `vault-value-out-of-list` | a wiki page with `status: later`; another with `reviewed:` empty | one finding: `status must be one of draft, done, found "later"` |
| `vault-unchecked-inbox` | `00-inbox/capture.md` without frontmatter | `[]` |

```go
// fixtures_test.go
func TestFixtures(t *testing.T) { /* walks testdata/fixtures, one subtest per folder, harness above */ }

// cli/check_test.go
func TestCheckText(t *testing.T) {
    // valid home tree → code 0, out == "OK: 3 pages\n"
    // tree of frontmatter-none → code 1, out ==
    //   "frontmatter: knowledge/glossary/contract.md: no frontmatter\nFAIL: 1 finding(s)\n"
}
func TestCheckSorted(t *testing.T) { /* two findings on b.md and one on a.md → a.md first; same path → check name order */ }
func TestCheckNoConfig(t *testing.T) { /* empty folder → code 2, errOut contains "no wikai.yaml" */ }
func TestCheckOutsideGit(t *testing.T) { /* valid tree, no repository → code 0, errOut contains "warning: not a git repository" */ }
```

- [ ] **Step 2: Run** `go test ./internal/check/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement** `check.go`, `stray.go`, `frontmatter.go`, `cli/check.go`, and `loadBase(env, cmd) (*wiki.Base, error)` in `cli/root.go` (`config.Locate` then `wiki.Load`).
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS (13 fixtures).
- [ ] **Step 5: Commit** — `git commit -m "feat: check command with stray and frontmatter checks"`

---

### Task 6: `links` check

**Files:**
- Create: `internal/check/links.go`; fixtures under `internal/check/testdata/fixtures/`
- Modify: `internal/check/check.go` (register)

**Interfaces:**
- Produces: `func Links(b *wiki.Base) []wiki.Finding`

Messages, all prefixed `<path>:<line>: `: `unparsed link` · `link leaves the base: <target>` · `target not found: <target>` · `anchor not found: <target>`.

Rules: read the links of every page and every doc. An `OutsideOnly` doc reports only `link leaves the base`. Markdown: after a resolved target, an anchor on a `.md` target must be in `wiki.Anchors` of the target text. Wikilink, on a `.md` target: an anchor that starts with `^` is checked with `wiki.HasBlockID`; any other anchor is a heading anchor and must equal one entry of `wiki.Headings` of the target, compared trimmed, in NFC, without case. For a nested anchor (`A#B`), only the last part is compared. The index doc is skipped when `index.check` is on and `other_links` is `forbid`: the `index` check then owns every index link (the ported fixture `index-dead-entry` expects one finding, from `index`).

- [ ] **Step 1: Port and write the fixtures**

Port: `anchor-repeated-heading`, `link-broken-anchor`, `link-broken-file`, `link-outside-home` (renamed `link-outside-base`), `link-wrong-case`.

New (vault config of Task 5):

| Fixture | Tree | Expected |
|---|---|---|
| `wikilink-valid` | links by name, by path suffix, with `\|text`, `![[img.png]]`, `[[Note#^b1]]` with the block present, `[[Note#Heading]]` with the heading present, a link in frontmatter `sources`, a link to a file whose name is NFD on disk, `[[Missing]]` inside a code fence and inside a code span | `[]` |
| `wikilink-broken` | `[[Missing]]` in the body of `20-wiki/concepts/A.md` | `links` · that path · `target not found: Missing` |
| `wikilink-block-missing` | `[[B#^nope]]`, `B.md` has no such block | `links` · `anchor not found` |
| `wikilink-heading-missing` | `[[B#Gone]]` and `[[B#faits datés]]`; `B.md` has `## Faits datés` only | one finding: `anchor not found: B#Gone` |
| `archive-link-outside` | home tree; `archive/old.md` holds a broken link and a link `../../../x.md` | one finding on the archive file: `link leaves the base` |

- [ ] **Step 2: Run** `go test ./internal/check/` — Expected: FAIL on the 10 new fixtures.
- [ ] **Step 3: Implement `Links`.**
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS (23 fixtures).
- [ ] **Step 5: Commit** — `git commit -m "feat: links check for markdown links and wikilinks"`

---

### Task 7: `index`, `duplicates`, `orphans` checks

**Files:**
- Create: `internal/check/index.go`, `duplicates.go`, `orphans.go`; fixtures
- Modify: `internal/check/check.go` (register)

**Interfaces:**
- Produces: `func Index(b)`, `func Duplicates(b)`, `func Orphans(b)`, each `[]wiki.Finding`; and the shared helper `func scope(b *wiki.Base, folders []string) []*wiki.Page` (pages under these root-relative folders; nil → every type folder).

Messages and rules:

- `index` (off when `Index` is nil or `Check` is false): `<index file> is missing` on the index path · `index link resolves to no page: <target>` on the index path (with `other_links: forbid`, for every link that does not resolve to a page of the scope; with `allow`, never) · `page is not linked from the index` · `page is linked <n> times from the index`. Scope = `scope(b, Index.Include)` minus the index itself.
- `duplicates` (always on, every page): `no title` (only with `title: heading`) · `name shared with another page: <names, sorted, comma-separated>`. Names of a page = its title and each entry of its `aliases` key (a list, or one non-empty text). Compare trimmed, NFC, lower case.
- `orphans` (off when `Orphans` is nil): `no other page links to this page`. Candidates = `scope(b, Orphans.Scope)` minus pages whose type is in `ExemptTypes`. A link counts when it comes from another page, and that page is not the index, not the log file, not in `IgnoreLinksFrom` (root-relative paths), and its type is not in `IgnoreLinksFromTypes`.

- [ ] **Step 1: Port and write the fixtures**

Port: `index-dead-entry`, `index-linked-twice`, `index-missing-page`, `duplicate-alias`, `duplicate-title`, `orphan-page`.

New:

| Fixture | Tree | Expected |
|---|---|---|
| `vault-orphan-exclusions` | Brain `orphans` block; page `A` linked only from `index.md`, `log.md` and a `type: mission` page; a mission page with no inbound link; page `B` linked from `A` | one finding: `orphans` on `A` |
| `vault-index-off` | `index: { file: index.md, check: false }`; a wiki page absent from the index | `[]` (the page is linked from another page) |
| `index-other-links-allow` | home tree with `other_links: allow`; the index links `README.md` | `[]` |
| `vault-duplicate-filename` | `20-wiki/a/Note.md` and `30-notes/note.md` | two `duplicates` findings |

- [ ] **Step 2: Run** `go test ./internal/check/` — Expected: FAIL on the 10 new fixtures.
- [ ] **Step 3: Implement** the three checks.
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS (33 fixtures).
- [ ] **Step 5: Commit** — `git commit -m "feat: index, duplicates and orphans checks"`

---

### Task 8: `sources` and `templates` checks

**Files:**
- Create: `internal/check/sources.go`, `templates.go`; fixtures
- Modify: `internal/check/check.go` (register)

**Interfaces:**
- Produces: `func Sources(b)`, `func Templates(b)`, each `[]wiki.Finding`.

`sources` (off when `Citations` is nil; pages whose `Citations.Frontmatter` key is a list):

- `<path>:<line>: unparsed citation`
- `<path>:<line>: cited but not in sources: <key>`
- `in sources but never cited: <entry>`
- With `code: true`, for a key that matches `^[A-Za-z0-9._-]+@[0-9A-Fa-f]+:`, checked against `^[A-Za-z0-9._-]+@[0-9a-f]{7}:[^#\s]+(?:#L(\d+)(?:-L(\d+))?)?$`:
  `<path>:<line>: malformed code citation: <key> (expected <repo>@<sha7>:<path>#L<start>-L<end>)` · `… code citation without a line range: <key>` · `… line range starts at 0: <key>` · `… reversed line range: <key>`. A valid code key is compared to the list without its `#L…` part.
- With `code: false`, a code-shaped key is plain text: no shape finding, no range removal.

`templates` (off when `TemplateMarkers` is empty): one finding per marker per page, `template marker left: <marker>`. Searched text = the frontmatter text plus the body outside code. For a `Frozen` page, the frontmatter text only.

- [ ] **Step 1: Port and write the fixtures**

Port: `code-citation-range-listed`, `code-citation-reversed-range`, `code-citation-without-lines`, `sources-not-cited`, `sources-not-listed`.

New:

| Fixture | Tree | Expected |
|---|---|---|
| `code-citation-off` | home tree with `citations: { frontmatter: sources, code: false }`; a page cites `(source: example-backend@1a2b3c4:a.md)` and lists the same text | `[]` |
| `citation-as-link` | a page cites `(source: [spec](../../docs/spec.md))`, lists `../../docs/spec.md`, and the file exists | `[]` |
| `vault-template-marker` | `template_markers: ["{{", "<%"]`, `frozen: [{path: 10-raw}]`; a wiki page with `{{todo}}` in the body and `<% x %>` in a code fence; a raw page with `{{x}}` in the body; a raw page with `{{x}}` in the frontmatter | two findings: the wiki page (`{{`) and the second raw page (`{{`) |

- [ ] **Step 2: Run** `go test ./internal/check/` — Expected: FAIL on the 8 new fixtures.
- [ ] **Step 3: Implement** the two checks.
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS (41 fixtures).
- [ ] **Step 5: Commit** — `git commit -m "feat: sources and templates checks"`

---

### Task 9: `frozen` check

**Files:**
- Create: `internal/check/frozen.go`; fixtures
- Modify: `internal/check/check.go` (`Run` calls it), `internal/check/fixtures_test.go` (count assertion)

**Interfaces:**
- Consumes: `gitx.BaseRef`, `gitx.LsTree`, `gitx.Show`.
- Produces: `func Frozen(b *wiki.Base, ref string) ([]wiki.Finding, error)` — `ref ""` returns nothing.

Rules: for each file that the tree of `ref` holds under a frozen path and that no `except` entry covers: absent from the working tree → `frozen file deleted`; different bytes → `frozen file modified`. A new file is valid. For each `append_only` file present at `ref`: absent → `append-only file deleted`; working content that does not start with the `ref` content → `append-only file rewritten`. A `ref` that git cannot read is an error (exit 2), message `cannot read git ref "<ref>"`.

- [ ] **Step 1: Port and write the fixtures**

Port: `archive-deleted`, `archive-modified`, `log-deleted`, `log-rewritten` (check name `frozen`).

New (Brain `frozen` and `append_only` blocks):

| Fixture | `base/` → `tree/` | Expected |
|---|---|---|
| `frozen-added` | a new file under `10-raw/` | `[]` |
| `frozen-except` | `10-raw/zotero/lib.md` and `10-raw/AGENTS.md` modified | `[]` |
| `frozen-renamed` | `10-raw/a.md` renamed `10-raw/b.md` | `frozen` · `10-raw/a.md` · `frozen file deleted` |
| `append-only-grown` | one line added at the end of `_meta/manifest.jsonl` | `[]` |
| `append-only-rewritten` | the first line of `_meta/manifest.jsonl` changed | `frozen` · `_meta/manifest.jsonl` · `append-only file rewritten` |

Add to `TestFixtures`: the folder holds 50 fixtures, 29 of them ported (`ported.txt` in `testdata/` lists the 29 names; the test reads it and fails when one is absent).

```go
// internal/cli/check_test.go
func TestCheckBaseFlag(t *testing.T) {
    // commit 1 adds knowledge/log/2026-01-01-init-x.md; commit 2 modifies it
    // "check"               → code 0 (base is HEAD)
    // "check --base HEAD~1" → code 1, out contains "frozen: knowledge/log/2026-01-01-init-x.md: frozen file modified"
    // "check --base nope"   → code 2
}
```

- [ ] **Step 2: Run** `go test ./internal/check/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement** `Frozen` and call it from `Run`.
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS (50 fixtures).
- [ ] **Step 5: Commit** — `git commit -m "feat: frozen and append-only check"`

---

### Task 10: `wikai new page`

**Files:**
- Create: `internal/write/slug.go`, `index.go`, `page.go`, `locale.go`, `internal/cli/new.go`
- Test: `internal/write/slug_test.go`, `index_test.go`, `page_test.go`, `internal/cli/new_test.go`

**Interfaces:**
- Consumes: `wiki.Base`, `config.KindOf`, `exit`.
- Produces:
  ```go
  func Slugify(s string) string   // NFKD, drop non-ASCII, lower, runs of non [a-z0-9] → "-", trim "-"
  type Text struct { EmptySection, EmptyGroup, MoveTitle, MoveNote string } // MoveTitle "Page %s renamed %s"
  func Locale(name string) Text
  // index.go — all three return the new index text; Usage error when the group heading is absent
  func IndexAdd(b *wiki.Base, index, pagePath, title, summary string) (string, error)
  func IndexRemove(b *wiki.Base, index, pagePath string) (string, error)
  func IndexResort(b *wiki.Base, index, pagePath string) (string, error)
  func LinkTo(b *wiki.Base, fromFile, pagePath, text string) string // "[text](rel/path.md)" or "[[stem]]"
  type PageOpts struct { Kind, Title, Summary, Slug string; Date time.Time; Set map[string]string }
  func NewPage(b *wiki.Base, o PageOpts) (path string, err error)
  ```
- CLI: `wikai new page <kind> <title> --summary <text> [--slug S] [--date D] [--set key=value]...`. Output: `created <path> and its line in <index path>` (without the second half when the page has no index line), then `next: fill the sections, link the page from another page, add the log entry (wikai new log)`. `--date` must be a real `YYYY-MM-DD`, else usage error; default `env.Now()`.

Rules:

- Locale texts: `en` → `Not documented.`, `No page.`, `Page %s renamed %s`, `renamed from %s`; `fr` → `Non renseigné.`, `Aucune page.`, `Page %s renommée %s`, `renommé depuis %s`.
- File: `<kind folder>/<name>.md`; name variables `{title}`, `{slug}`, `{date}` (`2026-01-02`), `{date:compact}` (`20260102`). An empty slug is a usage error (`the title gives an empty slug: pass --slug`). A name that holds `/`, `\`, or is `.`/`..` after expansion is a usage error.
- Refused (exit 1): `<path> exists`; `"<title>" is already the title or an alias of <path>: update that page` (same comparison as `duplicates`).
- Skeleton: `<skeletons>/<kind.skeleton, else the kind name>.md`; a missing skeleton is a usage error. A kind without a folder (its type has no `folders`) is a usage error: `kind <name> has no folder`.
- Fill, in this order: `{{title}}`, `{{date}}`, `{{slug}}`, `{{summary}}`, `{{key}}` of each `--set`; then each `set` value of the kind (replace the value of the frontmatter line `^key:`, or add the line before the closing `---`); then each empty frontmatter line `^key:[ \t]*$` whose rule is `required`, `format: date` and not `empty_allowed` receives the date; then each `## ` heading directly followed by another `## ` or the end of the file receives a blank line and the empty-section text.
- Index (when the index is on, or `check: false`, and the page is in `index.include`): group of a page = `## <folder>` for `groups: folder`, else `## <heading>` of the longest `groups` key that is a prefix of the page folder. The line is `- <link> — <summary>`. On insert: drop the empty-group text, keep non-entry lines first, sort entry lines by link target (`markdown`: byte order of the path; `wikilink`: NFC lower-case name). An emptied group receives the empty-group text.
- Order of writes: compute the page text and the new index text first; write the page with create-exclusive; then the index. A missing group fails before anything is written.

- [ ] **Step 1: Write the failing tests**

Home tree = the `valid` fixture of Task 5 with an index that has groups, plus `_skeletons/{term,decision,fact,design,repo}.md` written with `{{title}}`, e.g. `decision.md`: `---\ntype: decision\ncreated:\nupdated:\nsources: []\nsupersedes:\nsuperseded-by:\n---\n\n# {{title}}\n\n## Statement\n\n## Reason\n`.

```go
func TestSlugify(t *testing.T) {
    // "Éléments de l'API — v2" → "elements-de-l-api-v2";  "a.b_c" → "a-b-c";  "—" → ""
}
func TestNewPageHome(t *testing.T) {
    // NewPage{Kind:"decision", Title:"Use one binary", Summary:" One tool. ", Date:2026-01-02}
    // → "knowledge/decisions/adr-20260102-use-one-binary.md", content ==
    // "---\ntype: decision\ncreated: 2026-01-02\nupdated: 2026-01-02\nsources: []\nsupersedes:\nsuperseded-by:\n---\n\n# Use one binary\n\n## Statement\n\nNot documented.\n\n## Reason\n\nNot documented.\n"
    // index group "## decisions" holds "- [Use one binary](decisions/adr-20260102-use-one-binary.md) — One tool."
    // kind "fact" → fact-20260102-….md;  kind "design" → designs/2026-01-02-….md;  Slug:"x" → …-x.md
}
func TestNewPageSet(t *testing.T) {
    // repo skeleton holds "origin: {{origin}}"; Set{"origin":"git@example.com:acme/example-backend.git"} fills it
    // a "{{other}}" that nothing fills stays in the page
}
func TestIndexGroup(t *testing.T) {
    // group "## glossary\n\nNo page.\n" + first entry → the text "No page." is gone
    // entries stay sorted by path after an insert in the middle
    // index without "## repos" + a repo page → Usage error, and the page file does not exist
}
func TestNewPageRefused(t *testing.T) {
    // existing file → Refused containing "exists"
    // Title "CONTRACT" when a page is titled "contract" → Refused naming knowledge/glossary/contract.md
    // Title equal to an alias of another page → Refused
}
func TestNewPageVault(t *testing.T) { // brain.yaml of Task 2, skeleton wiki-entite.md with "kind:" and "created: {{date}}"
    // NewPage{Kind:"entity", Title:"Acme Tool", Summary:"A tool."} → "20-wiki/entites/Acme Tool.md", frontmatter line "kind: entity"
    // NewPage{Kind:"summary", Title:"Talk"} → "20-wiki/resumes/Résumé - Talk.md"
    //   index line "- [[Résumé - Talk]] — …" under "## Résumés de sources (`20-wiki/resumes/`)", sorted without case
    // the "reviewed:" line stays empty (empty_allowed)
    // a page outside index.include gets no index line
    // Title "a/b" → Usage error; no file written
}
func TestNewPageCLI(t *testing.T) {
    // run from knowledge/glossary: "new page term Offer --summary x --date 2026-01-02" → code 0,
    //   out line 1 == "created knowledge/glossary/offer.md and its line in knowledge/index.md"
    // without --summary → 2;  "--date 2026-02-30" → 2;  unknown kind → 2;  "--set novalue" → 2
}
```

- [ ] **Step 2: Run** `go test ./internal/write/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: new page command with index maintenance"`

---

### Task 11: `wikai new log`

**Files:**
- Create: `internal/write/log.go`
- Modify: `internal/cli/new.go`
- Test: `internal/write/log_test.go`, `internal/cli/new_test.go`

**Interfaces:**
- Produces:
  ```go
  type LogOpts struct { Operation, Title, Slug string; Created, Modified, Notes []string; Date time.Time }
  func NewLog(b *wiki.Base, o LogOpts) (path string, err error) // the entry file, or the log file
  ```
- CLI: `wikai new log <operation> <title> [--slug S] [--created P]... [--modified P]... [--note T]... [--date D]`. Output: `created <path>` (`dir`) or `appended to <path>` (`file`).

Rules: a nil `Log` block, an operation outside `operations`, an entry without any item, or an empty slug in `dir` mode are usage errors. Header `## [<date>] <operation> | <title>`. Then one line per item, `- <label><separator><item>`, in the order created, modified, note. With `one_line`, the created items share one line joined by `, `, and so do the modified items; each note keeps its own line. With `as_links`, a created or modified item that is a page path becomes `LinkTo` from the log file (text = the page title); any other item is written as given (`wikilink`: `[[item]]`). `dir`: file `<log.path>/<date>-<operation>-<slug>.md`, create-exclusive; an existing file is Refused: `<path> exists: log entries are never modified, choose another --slug`. `file`: the log file must exist (usage error otherwise); the entry is appended after one blank line.

- [ ] **Step 1: Write the failing tests**

```go
func TestNewLogDir(t *testing.T) {
    // {Operation:"ingest", Title:"Billing notes", Created:["knowledge/glossary/offer.md"], Modified:["knowledge/index.md"], Notes:["first","second"], Date:2026-01-02}
    // → "knowledge/log/2026-01-02-ingest-billing-notes.md" ==
    // "## [2026-01-02] ingest | Billing notes\n- created: knowledge/glossary/offer.md\n- modified: knowledge/index.md\n- note: first\n- note: second\n"
    // second call, same arguments → Refused, file unchanged
    // Operation "deploy" → Usage;  no item → Usage
}
func TestNewLogFile(t *testing.T) { // brain.yaml; log.md ends with "…\n- note : old\n"
    // {Operation:"ingest", Title:"Talk", Created:["20-wiki/resumes/Résumé - Talk.md","20-wiki/concepts/Evals.md"], Modified:["index.md"], Notes:["done"], Date:2026-01-02}
    // → log.md ends with
    // "- note : old\n\n## [2026-01-02] ingest | Talk\n- créé : [[Résumé - Talk]], [[Evals]]\n- modifié : [[index]]\n- note : done\n"
    // everything before the appended entry is byte-identical
}
```

- [ ] **Step 2: Run** `go test ./internal/write/` — Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: new log command"`

---

### Task 12: `wikai move`

**Files:**
- Create: `internal/write/move.go`, `internal/cli/move.go`
- Test: `internal/write/move_test.go`, `internal/cli/move_test.go`

**Interfaces:**
- Consumes: `wiki.RewriteLinks`, `gitx.Mv`, `IndexRemove`, `IndexResort`, `NewLog`, `Locale`.
- Produces:
  ```go
  type MoveOpts struct { Old, New string; NoLog bool; Notes []string; Date time.Time } // boundary-relative paths
  type MoveResult struct { Modified []string; Log string }
  func Move(b *wiki.Base, o MoveOpts) (MoveResult, error)
  ```
- CLI: `wikai move <old> <new> [--no-log] [--note T]... [--date D]`. Output: `moved <old> to <new>; links rewritten in <n> page(s); index updated`, then `created <log>` or `appended to <log>`.

Rules:

- Usage errors: `<old> is not a page`; `<new>` does not end with `.md` or is outside `root`; not a git repository.
- Refused: `<new> exists`; the log entry file exists (`dir` mode, checked before `git mv`); the folder of `<new>` is neither a folder of the page's type nor a frozen path: `a page moves only inside its type or to a frozen path`.
- Then `git mv`. Then rewrite in every page and in the index doc:
  - `markdown`: a link that resolved to `<old>` receives the relative path to `<new>` from the linking file; the `#anchor` is kept.
  - `wikilink`, only when the file stem changes: a link that resolved to `<old>` receives the new stem (or the new root-relative path without `.md` when the link held a `/`); `#anchor` and `|text` are kept. Frontmatter links are rewritten too.
  - The moved page itself, `markdown` style, when its folder changes: each of its relative links is recomputed from the new folder, so each link keeps its target.
- Index: the group of the page is re-sorted; a move to a frozen path removes the line (an archive).
- Log (unless `--no-log` or no `Log` block): operation `move`, title `MoveTitle(old, new)`, slug `Slugify(stem of new)`, created `[new]`, modified = the sorted rewritten pages (the index excluded), notes = `MoveNote(old)` then `o.Notes`.

- [ ] **Step 1: Write the failing tests**

```go
func TestMoveRename(t *testing.T) { // home tree, committed
    // Move knowledge/glossary/contract.md → knowledge/glossary/agreement.md, Date 2026-01-03
    // api-contract.md now holds "(../glossary/agreement.md#definition)"; the index line target is glossary/agreement.md and the group is sorted
    // Modified == ["knowledge/domains/api-contract.md"]
    // log file knowledge/log/2026-01-03-move-agreement.md ==
    // "## [2026-01-03] move | Page knowledge/glossary/contract.md renamed knowledge/glossary/agreement.md\n- created: knowledge/glossary/agreement.md\n- modified: knowledge/domains/api-contract.md\n- note: renamed from knowledge/glossary/contract.md\n"
    // `git status --porcelain` shows the rename as staged (R)
    // a link to contract.md inside a code fence is unchanged
}
func TestMoveToArchive(t *testing.T) {
    // a term page → knowledge/archive/contract.md: accepted (any type); its index line is gone;
    // a link "../domains/api-contract.md" inside the moved page still resolves to the same file
}
func TestMoveRefused(t *testing.T) {
    // glossary/x.md → repos/x.md: Refused;  existing target: Refused;  existing log entry: Refused
    // in each case the old file still exists and `git status --porcelain` is empty
    // knowledge/index.md as <old>: Usage;  target "../x.md": Usage
}
func TestMoveVault(t *testing.T) { // brain.yaml, locale fr
    // "20-wiki/concepts/Evals.md" → "20-wiki/concepts/LLM evals.md"
    // "[[Evals]]" → "[[LLM evals]]";  "[[Evals#^b1|les evals]]" → "[[LLM evals#^b1|les evals]]"
    // "[[concepts/Evals]]" → "[[20-wiki/concepts/LLM evals]]";  the frontmatter link is rewritten
    // index line "- [[LLM evals]] — …" re-sorted in its group
    // log.md ends with "## [<date>] move | Page 20-wiki/concepts/Evals.md renommée 20-wiki/concepts/LLM evals.md\n- créé : [[LLM evals]]\n- modifié : …\n- note : renommé depuis 20-wiki/concepts/Evals.md\n"
}
func TestMoveNoLog(t *testing.T) { /* --no-log → Log == "" and the log folder is unchanged */ }
func TestMoveCLIFromSubfolder(t *testing.T) { /* cwd knowledge/glossary: "move contract.md agreement.md" → code 0 */ }
```

- [ ] **Step 2: Run** `go test ./internal/write/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: move command with link, index and log updates"`

---

### Task 13: `wikai manifest`

**Files:**
- Create: `internal/manifest/manifest.go`, `internal/cli/manifest.go`
- Test: `internal/manifest/manifest_test.go`, `internal/cli/manifest_test.go`

**Interfaces:**
- Produces:
  ```go
  type Record struct {
      TS, Raw, SHA256, Status, Pipeline string
      Pages []string
      Note  string
  }
  func Normalize(text string) string
  func Hash(file string) (string, error)
  func Status(manifestFile, raw, digest string) (string, error)
  func Append(file string, r Record) (line string, err error)   // file lock, returns the written line
  func (r Record) Line() string
  ```
- CLI: `wikai manifest hash|status|append <note> [--status S] [--pages P...] [--note T]`. `hash` prints the digest. `status` prints `new`, `unchanged <status> <raw>` or `modified`. `append` prints the written line. `raw` is the boundary-relative path of the note. A nil `Manifest` block, a missing `--status` on `append`, or a status outside `statuses` are usage errors.

`Normalize`, in this order (port of `manifest.py`; the edge behaviours are intended):

1. Remove every leading U+FEFF. Replace `\r\n` then `\r` with `\n`.
2. When the text starts with `---\n`: find the first `\n---` at or after byte index 3, and drop everything up to and including these four bytes. The rest of that line stays.
3. Remove `[ \t]\^[A-Za-z0-9-]+[ \t]*` at each line end.
4. Merge each whitespace run into one space and trim. Whitespace here is Python's `str` whitespace: `unicode.IsSpace` plus U+001C–U+001F. Go's `\s` is ASCII only and gives wrong hashes.

`Line`: Go's compact JSON (no space after `,` and `:`), keys in the order `ts, raw, sha256, status, pipeline, pages, note`, non-ASCII characters and `<>&` written as they are (`SetEscapeHTML(false)`), `pages` is `[]` when empty. The lines `manifest.py` wrote keep their spaced form and are never rewritten; `Status` reads both forms. `ts` is UTC, `2006-01-02T15:04:05Z`.

`Status`: the last record with the same digest → `unchanged <its status> <its raw>`; else any record with the same `raw` → `modified`; else `new`. Blank lines of the manifest are skipped; an unreadable line is an error that names its line number.

- [ ] **Step 1: Write the failing tests**

Reference values, produced by `manifest.py` on 2026-11-25:

```go
var golden = []struct{ in, norm, sha string }{
    {"Hello  world\n\nSecond line.\n", "Hello world Second line.",
        "411aaaca8e6dc12957cbd87d69ee340f0c2efa5859ba8a461b9a215fcb1c68cf"},
    {"﻿Hello\r\nworld\r\n", "Hello world",
        "64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c"},
    {"---\ntype: source\ntags: []\n---\n# Title\n\nBody text.\n", "# Title Body text.",
        "5006560973a1a5576d49014be4376537585edaf531cd6745f304c351d1e6bb1f"},
    {"A quoted line ^abc-123\nAnother line. ^q2  \nNo id here ^ not-an-id\n",
        "A quoted line Another line. No id here ^ not-an-id",
        "4e68ae6304210a303457b012af36a52163086cbec0c83853bbd1e5dce150d628"},
    {"café noir fin\n", "café noir fin",
        "584e789766a0db33676704d7485a78f300eaeb80161fcb2baf09b8a3c8baa6b1"},
    {"---\na: 1\n--- trailing\nBody\n", "trailing Body",
        "d24aed7af3ed9be2ad5cafa014e43173f79e87ea1c7f80536373f126be5cc6a8"},
}
func TestNormalizeAndHash(t *testing.T) { /* Normalize(in) == norm; sha256 hex of norm == sha; Hash(file with in) == sha */ }
func TestLine(t *testing.T) {
    r := Record{TS: "2026-01-02T03:04:05Z", Raw: "10-raw/web/Café <a&b>.md", SHA256: "ab", Status: "integrated",
        Pipeline: "ingest-v2", Pages: nil, Note: ""}
    // r.Line() == `{"ts":"2026-01-02T03:04:05Z","raw":"10-raw/web/Café <a&b>.md","sha256":"ab","status":"integrated","pipeline":"ingest-v2","pages":[],"note":""}`
    // with Pages ["A", "B"] → `"pages":["A","B"]`
}
func TestStatus(t *testing.T) {
    // empty or absent manifest → "new"
    // a manifest that holds one line in the spaced form of manifest.py (`{"ts": "…", "raw": "10-raw/n.md", …}`) with the digest → "unchanged integrated 10-raw/n.md"
    // after Append(status integrated) → "unchanged integrated 10-raw/n.md"
    // note content changed → "modified"
    // same content under another path → "unchanged integrated 10-raw/n.md"
    // two records with the digest → the status of the last one
}
func TestAppendConcurrent(t *testing.T) { /* 20 goroutines append; the file has 20 lines; each line parses as JSON */ }
func TestManifestCLI(t *testing.T) {
    // "manifest append 10-raw/n.md" (no --status) → 2;  "--status nope" → 2
    // "manifest append 10-raw/n.md --status integrated --pages A B --note ok" → 0, out is the line, the file ends with it
    // run from 10-raw: "manifest status n.md" → "unchanged integrated 10-raw/n.md"
    // a base without manifest block → 2
}
```

- [ ] **Step 2: Run** `go test ./internal/manifest/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement.** Lock with `syscall.Flock(fd, syscall.LOCK_EX)` on the file opened with `O_APPEND|O_CREATE|O_WRONLY`; create the parent folder.
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: manifest hash, status and append"`

---

### Task 14: `wikai guard hook` and `wikai guard commit`

**Files:**
- Create: `internal/guard/rules.go`, `hook.go`, `commit.go`, `internal/cli/guard.go`
- Test: `internal/guard/rules_test.go`, `hook_test.go`, `commit_test.go`, `internal/cli/guard_test.go`

**Interfaces:**
- Consumes: `config.Config`, `gitx.Diff`, `gitx.Show`, `gitx.MergeBase`.
- Produces:
  ```go
  type Rule int
  const ( None Rule = iota; AppendOnly; Frozen; Protected )
  func Classify(cfg *config.Config, path string) Rule   // path is boundary-relative
  type Event struct {
      ToolName  string `json:"tool_name"`
      Cwd       string `json:"cwd"`
      ToolInput struct {
          FilePath     string `json:"file_path"`
          NotebookPath string `json:"notebook_path"`
          Content      string `json:"content"`
      } `json:"tool_input"`
  }
  func Hook(cfg *config.Config, boundary string, ev Event) (deny string) // "" allows
  type Violation struct { Status, Path, Reason string }
  func Commit(cfg *config.Config, boundary string, staged bool, base, head string, agent bool) ([]Violation, error)
  ```

Rules:

- `Classify`: `AppendOnly` when the path is an `append_only` file; else `Frozen` when it is under a frozen path and under none of its `except` entries; else `Protected` when it equals or is under a `protected` entry; else `None`. `.git/` is always `Protected`.
- `guard hook`: the start folder is `CLAUDE_PROJECT_DIR`, else the event `cwd`, else the current folder. No `wikai.yaml` above it → exit 0. Invalid `wikai.yaml` or stdin that is not a JSON object → exit 2 with the reason (fail closed). No target path → exit 0. The target is made absolute from the event `cwd`, then its longest existing parent is passed through `filepath.EvalSymlinks`. A target outside the boundary → exit 0.
  - `Protected` → deny `<path>: protected path, propose the change to the human`.
  - `Frozen` → deny when the file exists: `<path>: frozen file, existing files are never modified`.
  - `AppendOnly` → allow when the file does not exist, or when the tool is `Write` and `content` starts with the current content; else deny `<path>: append-only file, new content must start with the current content`.
  - A denial prints the message on stderr and exits 2.
- `guard commit`: `--staged` reads `gitx.Diff(dir, "--cached")`, old side `HEAD`, new side the index. `<base> <head>` reads `gitx.Diff(dir, base+"..."+head)`, old side the merge-base, new side `<head>`. Neither form, or both, is a usage error.
  - `Frozen` and status not `A` → `frozen file`.
  - `AppendOnly` and the old content is not a prefix of the new content (a deleted file included) → `append-only file: existing lines changed`.
  - With `--agent`: `Protected` → `protected path`.
  - No violation: stdout `guard: OK`, exit 0. Else stderr `guard: changes refused`, then one line `<status> <path>: <reason>` per violation, sorted by path; exit 1.

- [ ] **Step 1: Write the failing tests**

```go
func TestClassify(t *testing.T) { // brain.yaml of Task 2
    cases := map[string]Rule{
        "_meta/manifest.jsonl": AppendOnly, "_meta/lint/Run 1.md": Frozen, "_meta/schema.md": Protected,
        "10-raw/web/a.md": Frozen, "10-raw/zotero/lib.md": Protected, "10-raw/AGENTS.md": None,
        "CLAUDE.md": Protected, ".claude/settings.json": Protected, "wikai.yaml": Protected,
        ".git/config": Protected, "20-wiki/concepts/A.md": None, "_metadata.md": None,
    }
}
func TestHook(t *testing.T) { // vault on disk with 10-raw/web/a.md, _meta/lint/Run 1.md, _meta/manifest.jsonl ("L1\n")
    // Write 10-raw/web/new.md                          → allow
    // Edit  10-raw/web/a.md                            → deny containing "frozen"
    // Write _meta/lint/Run 2.md                        → allow;  Write _meta/lint/Run 1.md → deny
    // Write _meta/manifest.jsonl content "L1\nL2\n"    → allow;  content "X\n" → deny;  Edit → deny
    // Write _meta/schema.md                            → deny containing "protected"
    // NotebookEdit with notebook_path under 10-raw/zotero/ → deny
    // file_path "/elsewhere/x.md"                      → allow
    // relative file_path "CLAUDE.md" with cwd = vault  → deny
    // AGENTS.md is a symlink to CLAUDE.md; Write AGENTS.md → deny
}
func TestHookCLI(t *testing.T) {
    // stdin `{"tool_name":"Edit","tool_input":{"file_path":"<vault>/CLAUDE.md"},"cwd":"<vault>"}` → code 2, errOut contains "CLAUDE.md"
    // stdin "not json" → code 2
    // folder without wikai.yaml → code 0
    // wikai.yaml holding "colour: blue" → code 2
    // Getenv("CLAUDE_PROJECT_DIR") = vault while cwd is elsewhere → the vault config applies
}
func TestCommit(t *testing.T) { // repository with the vault committed
    // staged: new 10-raw/web/b.md → OK;  modified 10-raw/web/a.md → [{M, 10-raw/web/a.md, frozen file}]
    // staged: deleted 10-raw/web/a.md → D;  `git mv` of it → D on the old path only
    // staged: modified 10-raw/zotero/lib.md → OK;  with agent → protected path
    // staged: manifest grown → OK;  first line changed → append-only violation;  manifest deleted → violation
    // staged: modified _meta/schema.md → OK;  with agent → violation
    // base/head: same cases across two commits
}
func TestCommitCLI(t *testing.T) {
    // "guard commit --staged" clean → 0, out "guard: OK\n"
    // violation → 1, errOut "guard: changes refused\nM 10-raw/web/a.md: frozen file\n"
    // "guard commit" → 2;  "guard commit --staged a b" → 2
}
```

- [ ] **Step 2: Run** `go test ./internal/guard/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: guard hook and guard commit"`

---

### Task 15: `wikai init` and the two presets

**Files:**
- Create: `internal/preset/preset.go`, `internal/preset/home-repo/…`, `internal/preset/obsidian-vault/…`, `internal/cli/init.go`
- Test: `internal/preset/preset_test.go`, `internal/cli/init_test.go`

**Interfaces:**
- Produces:
  ```go
  func Names() []string                                   // ["home-repo", "obsidian-vault"]
  func Write(name, dir string, date time.Time) ([]string, error) // created paths; Refused when dir/wikai.yaml exists
  ```
- CLI: `wikai init [dir] --preset home-repo|obsidian-vault`. `dir` defaults to the current folder and is created when absent. Output: one `created <path>` line per file. A missing or unknown `--preset` is a usage error.

Each preset folder is embedded with `embed.FS` and copied as it is; `{{date}}` in a non-skeleton file is replaced by the date. Every type folder, the log folder and each frozen folder receive a `.gitkeep`. No agent contract and no procedure file is written.

- `home-repo`: `wikai.yaml` = the home example of the spec plus `domain: { folders: [domains] }` and `ignore: [README.md]`. Files under `knowledge/`: `index.md` (`# Index`, then the groups `glossary`, `repos`, `domains`, `decisions`, `designs`, each with `No page.`), `_skeletons/{term,repo,domain,decision,fact,design}.md`, and the folders `glossary`, `repos`, `domains`, `decisions`, `designs`, `archive`, `log`. The `repo` skeleton holds `origin: {{origin}}`.
- `obsidian-vault`: `root: .`, `locale: en`, `title: filename`, `links: wikilink`, `stray: false`, `ignore: [.obsidian, _meta]`, `skeletons: _meta/skeletons`. Global frontmatter: `type` and `created` (date) required. Types: `wiki` (folders `[wiki]`; `status` in `[draft, done]`; `sources` list; kinds `concept` → `wiki/concepts`, `summary` → `wiki/summaries` named `Summary - {title}`, `entity` → `wiki/entities`, each with `set: { kind: <name> }`), `source` (folders `[raw]`), `note` (folders `[notes]`), `meta: {}`. Index `index.md` with `include: [wiki]` and three group headings; log `mode: file`, `path: log.md`, operations `[init, ingest, query, lint, move]`, `one_line` and `as_links` on. `frozen: [{ path: raw }]`, `append_only: [_meta/manifest.jsonl]`, `template_markers: ["{{", "<%"]`, `orphans: { scope: [wiki], ignore_links_from: [index.md, log.md] }`, `manifest: { file: _meta/manifest.jsonl, pipeline: ingest-v1, statuses: [integrated, no_material, error, superseded] }`, `protected: [_meta, wikai.yaml]`. `index.md` and `log.md` carry a frontmatter `type: meta`, `created: {{date}}`.

- [ ] **Step 1: Write the failing tests**

```go
func TestPresetsLoad(t *testing.T) { /* for each name: Write into a temp dir; config.Load(dir/wikai.yaml) succeeds */ }
func TestInitThenCheck(t *testing.T) {
    // for each preset: "init <dir> --preset <name>" → 0; git init + add -A in dir; "check" → 0
    // home-repo → out "OK: 0 pages\n";  obsidian-vault → out "OK: 2 pages\n"
}
func TestInitEveryKindCreates(t *testing.T) {
    // for each preset and each kind of its config that has a folder: "new page <kind> T-<kind> --summary s" (plus "--set origin=git@example.com:acme/x.git" for repo) → 0
    // then `git add -A` and "check --json": every finding has check "orphans"
}
func TestInitRefusesExisting(t *testing.T) { /* second init in the same dir → 1; wikai.yaml unchanged */ }
func TestInitUsage(t *testing.T)           { /* no --preset → 2; "--preset nope" → 2 */ }
```

- [ ] **Step 2: Run** `go test ./internal/preset/ ./internal/cli/` — Expected: FAIL.
- [ ] **Step 3: Implement** and write the preset files.
- [ ] **Step 4: Run** `go test ./...` — Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: init command with home-repo and obsidian-vault presets"`

---

### Task 16: Distribution

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh`, `README.md`, `LICENSE`
- Modify: `.github/workflows/gates.yml`, `.github/workflows/ci.yml`

Prerequisites for the human (the release cannot run without them):

1. The repository `github.com/PillowPillow/wikai` exists and this branch is pushed.
2. The repository secret `TAP_GITHUB_TOKEN` is set, with write access to `PillowPillow/homebrew-tap` (the same token `den` uses).

`LICENSE` is the GPL-3.0 text (the licence of `github.com/PillowPillow/tea-shot`, approved by the human). Take the text from https://www.gnu.org/licenses/gpl-3.0.txt, unchanged.

- [ ] **Step 1: Write `.goreleaser.yaml`** from `~/Development/Pillow/den/.goreleaser.yaml`, with these values:

  - `project_name: wikai`; `release.github`: owner `PillowPillow`, name `wikai`; `header: "{{ .TagBody }}"`.
  - `builds`: `main: ./cmd/wikai`, `binary: wikai`, `CGO_ENABLED=0`, `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `ldflags: -s -w -X github.com/PillowPillow/wikai/internal/cli.Version={{ .Tag }}`.
  - `archives`: `tar.gz`, `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}`; `checksum: checksums.txt`; `changelog.disable: true`.
  - `homebrew_casks`: name `wikai`, repository `PillowPillow/homebrew-tap` with `token: "{{ .Env.TAP_GITHUB_TOKEN }}"`, homepage `https://github.com/PillowPillow/wikai`, description `Check, write and guard LLM wikis from one config file`, the same `url.template` shape, the same quarantine `custom_block` with `wikai`, the same `commit_author`.

- [ ] **Step 2: Add to `gates.yml`**, after the govulncheck step: `goreleaser/goreleaser-action@v7` with `distribution: goreleaser`, `version: "~> v2"`, `args: build --snapshot --single-target --clean`; then a step that finds the one `wikai` binary under `dist/`, runs `--version`, and fails unless the output equals `jq -r .tag dist/metadata.json`.

- [ ] **Step 3: Write `release.yml`** as in `den`: `on: push: tags: ["v*"]`, `permissions: contents: write`, job `test` uses `gates.yml`, job `release` needs `test`: checkout with `fetch-depth: 0`, `git fetch --force --tags`, setup-go from `go.mod`, goreleaser-action `release --clean` with `GITHUB_TOKEN` and `TAP_GITHUB_TOKEN`.

- [ ] **Step 4: Write `install.sh`** from `~/Development/Pillow/den/install.sh`: `den` → `wikai`, `DEN_INSTALL_DIR` → `WIKAI_INSTALL_DIR`, `DEN_VERSION` → `WIKAI_VERSION`; the version probe is `wikai --version`, which prints the tag alone. Add `shellcheck install.sh` as a step of `gates.yml`.

- [ ] **Step 5: Write `README.md`**: what `wikai` is (two sentences), `brew install --cask PillowPillow/tap/wikai`, the `install.sh` one-liner, one line per command, the `wikai.yaml` home example with a link to the spec for each block, the hook snippet for `.claude/settings.json` (`"command": "wikai guard hook"`, matcher `Edit|Write|NotebookEdit`), and the pre-commit line `exec wikai guard commit --staged`.

- [ ] **Step 6: Verify locally**

Run: `go build -ldflags "-X github.com/PillowPillow/wikai/internal/cli.Version=v0.0.0-test" -o wikai ./cmd/wikai && ./wikai --version`
Expected: `v0.0.0-test`

Run: `shellcheck install.sh && go vet ./... && test -z "$(gofmt -l .)" && go test -count=1 ./... && go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...`
Expected: PASS and `No vulnerabilities found.`

- [ ] **Step 7: Commit** — `git commit -m "build: goreleaser, release workflow, install script and README"`

- [ ] **Step 8 (human, after Task 17): release.** Warning: a pushed tag publishes a public release and a cask; it is not reversible. Tag with `git tag -a --cleanup=verbatim v0.1.0` and push the tag. Then run `brew install --cask PillowPillow/tap/wikai && wikai --version`. Expected: `v0.1.0`.

- [ ] **Step 9 (after the first release): install script smoke test.** Add to `ci.yml` the job `install-script` of `den`'s `ci.yml`, adapted: `WIKAI_INSTALL_DIR="$RUNNER_TEMP/bin" sh install.sh`, then `wikai --version` must equal the latest release tag; `WIKAI_VERSION=v9.9.9 sh install.sh` must fail and write no binary. This job needs a published release, so it cannot enter CI before Step 8. Commit — `git commit -m "ci: install script smoke test"`

---

### Task 17: Acceptance on the three bases

**Files:**
- Create: `acceptance/run.sh`
- Create, outside the repository: `brain.yaml`, `am-home.yaml`, `dev-home.yaml` in the folder that `WIKAI_ACCEPTANCE_CONFIGS` names (default `~/.config/wikai-acceptance`)

This task runs on the author's machine only; CI does not run it. It never writes into a base. The three configs hold the folder and type names of the bases: they stay on the machine and never enter this repository.

- [ ] **Step 1: Record the current reference.** Run the three current scripts and keep their last line:

```bash
python3 ~/Development/Pillow/am-home/knowledge/_ops/check.py check
python3 ~/Development/Digitaleo/dev-home/knowledge/_ops/check.py check
(cd ~/Brain && python3 _meta/scripts/health_check.py)
```

Expected: `OK: <n> pages` twice, and `Total : 0 problème(s)` with `Notes contrôlées : <n>`. (On 2026-11-25: 148, 15, 129.)

- [ ] **Step 2: Write the three configs** in the configs folder.

  - `am-home.yaml`: `internal/check/testdata/home.yaml`.
  - `dev-home.yaml`: the same with `locale: fr`, `base_branch: master`, `citations.code: false`.
  - `brain.yaml`: `internal/config/testdata/brain.yaml`, completed from the tables of `~/Brain/_meta/scripts/health_check.py`: the `COMMON` keys go to the top-level `frontmatter`; each entry of `SCHEMA` becomes a type with its required keys and its `status` values; `KINDS` are the `kind` values; `NOT_CHECKED` and `RULE_FILES` give `ignore`; `WIKI_DIRS` gives `orphans.scope`; `RAW_EXEMPT` gives the `except` list. The Python script checks no date format: set `format: date` on a key only when the run stays clean.

- [ ] **Step 3: Write `acceptance/run.sh`**: builds `./wikai`, then for each base runs `./wikai check --config "$WIKAI_ACCEPTANCE_CONFIGS/<name>.yaml" --dir <base path>`; base paths come from `WIKAI_BRAIN`, `WIKAI_AM_HOME`, `WIKAI_DEV_HOME` with the three default paths of the spec; the script exits non-zero when a run does, and when a config file is absent.

- [ ] **Step 4: Run** `sh acceptance/run.sh`. The criterion is strict: `OK: <n> pages` on the three bases, with the counts of Step 1, and zero finding. Sort every finding:

  - A page count that differs from Step 1, or a finding on a rule both tools share that the Python script does not report: a defect of `wikai` or of the config. Reproduce it as a fixture in `internal/check/testdata/fixtures/` (fictitious names), fix, rerun.
  - A finding from a rule that is new by design (a frontmatter the YAML parser refuses, a missing heading anchor in a Brain wikilink, a frozen file changed in an unpushed local commit): a defect of the base. Stop and report the list to the human. The human corrects the page in the base, or pushes the commit. This project never writes into a base.

  Any finding blocks the release.

- [ ] **Step 5: Verify that the bases are untouched.** Run `git -C <base> status --porcelain` on the three bases before Step 4 and after it. Expected: the same output.

- [ ] **Step 6: Commit** — `git commit -m "test: acceptance script for the three bases"`
