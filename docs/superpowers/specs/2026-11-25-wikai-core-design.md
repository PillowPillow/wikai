# wikai core — design

Date: 2026-11-25. Status: approved by checklist on 2026-11-25 (`2026-11-25-wikai-core-validation.md`); the checklist wins over this document. Skeleton date and time markers (v0.2.0): approved by checklist on 2026-11-26 (`2026-11-26-obsidian-templates-validation.md`).

## Intent

Three knowledge bases follow the LLM wiki pattern (immutable sources, agent-written pages, an index, a log, the ingest / query / lint procedures):

- **Brain**, an Obsidian vault;
- the `knowledge/` folder of **am-home**;
- the `knowledge/` folder of **dev-home**.

Their deterministic tooling is custom and duplicated: Brain has 402 lines of Python and sh without tests; each home has about 1,000 lines of Python plus tests, and the two copies already differ. `wikai` is one Go binary that replaces these scripts for any LLM wiki. A new wiki is set up with a config file and no code.

`wikai` is a tool for LLM wikis. Home repo commands are a secondary group, designed in project 2.

## The four projects

| # | Project | Document |
|---|---|---|
| 1 | Core wiki: config, `check`, `new`, `move`, `manifest`, `guard`, `init`, distribution | this one |
| 2 | Home repo commands: `wikai repo locate`, `wikai repo path`, home layout checks | later |
| 3 | Migration of dev-home, then am-home, then Brain; removal of the Python and sh scripts | later |
| 4 | Marketplace of generic skills for Claude Code and Codex | later |

## Goals

- One file, `wikai.yaml`, describes a base. Every command reads it; a rule is written once.
- `wikai check` reports what `check.py check` and `health_check.py` report today, on the three bases, without any change to their pages.
- `wikai new` and `wikai move` replace `new.py`, and give Brain the page, index and log creation it lacks.
- `wikai manifest` replaces `manifest.py`; existing manifest lines stay valid.
- `wikai guard` replaces `protect_raw.py` and `guard.sh`, and is available to the homes.
- `wikai init` creates a new base from a preset.
- Installed with Homebrew, on macOS and Linux.

## Non-goals of this project

- `repo locate`, `repo path`, home layout checks (project 2).
- Changing the three bases, their procedures or their CI (project 3).
- Skills and marketplace (project 4).
- Capture (`brain-ingest`: yt-dlp, transcription).
- Search and query.
- Any call to a model or to the network.

## Anti-goals

- A config that becomes a language: no expressions, no conditions, no user-defined checks. A rule that no mechanism covers stays a project check outside `wikai`.
- Requiring a base to rewrite its pages to fit the tool.
- Behaviour hidden behind a profile name: a preset is only a starting `wikai.yaml` that the user can read and edit.

## Approaches considered

1. **One convention.** The CLI imposes one format and the bases migrate. Rejected: Obsidian needs wikilinks; the homes need relative markdown links to render on GitHub and GitLab.
2. **Profiles coded in Go** (`profile: obsidian-vault`). Rejected by the human: a third kind of wiki needs a new release, and the rules live in the binary.
3. **Mechanisms set by config** — chosen by the human. The binary ships a fixed set of mechanisms; `wikai.yaml` turns them on and sets their parameters; `init` writes a starting config.

## Decided with the human

- Name: `wikai`.
- Scope of the whole effort: wiki and home repo, without capture.
- Approach 3.
- The config blocks: base location and link style, page types, index, log, frozen paths, provenance, locale.
- Frontmatter rules are written under a `frontmatter` key, one entry per key, not as a list of required keys.
- Order of the four projects.

Everything below that is not in this list is the author's call and is open to correction.

## Constraints

- Go, cobra, `gopkg.in/yaml.v3`, goreleaser, a Homebrew cask in `PillowPillow/tap`: the recipe of `den` (see its `.goreleaser.yaml` and `install.sh`).
- Zero known vulnerability. Go and every dependency are at their latest stable version, checked at the source when the plan is written and not from memory. `govulncheck ./...` runs in CI on every push and before a release; one finding fails the build. The dependency list stays as short as the code allows.
- Module `github.com/PillowPillow/wikai`, repository `git@github.com:PillowPillow/wikai.git`.
- git is the only external program. Without a git repository, the frozen-path check is skipped with a warning on stderr.
- Output is deterministic: findings sorted by path, then by check.
- CLI messages are English. `locale` only sets text that `wikai` writes into pages.
- Tests never contain a real client, person, repo or host name. Fixtures ported from the homes are rewritten with fictitious names.

## `wikai.yaml`

The file sits at the root of the base (the git repository root for a home, the vault root for Brain). Its folder is the **boundary**: no link may leave it. A command finds the file by walking up from the current folder. `--config <file>` reads another file; `--dir <folder>` then names the boundary (default with `--config`: the current folder). `root` and `protected` are relative to the boundary; every other path in the config is relative to `root`. Paths that `wikai` prints or records (findings, created files, the manifest `raw`) are relative to the boundary. Path arguments on the command line are resolved from the current folder. An unknown key is an error, not a warning.

Example for a home:

```yaml
version: 1
root: knowledge                # pages live under this folder; "." for a vault
locale: en                     # en | fr
title: heading                 # heading (first "# " line) | filename
links: markdown                # markdown | wikilink

frontmatter:                   # rules for every page
  type:    { required: true }
  created: { required: true, format: date }
  updated: { required: true, format: date }
  sources: { required: true, format: list }

types:
  term:
    folders: [glossary]
    frontmatter:
      aliases: { required: true, format: list }
  repo:
    folders: [repos]
    frontmatter:
      origin:  { required: true }
      aliases: { required: true, format: list }
  decision:
    folders: [decisions]
    frontmatter:
      supersedes:    { required: true }
      superseded-by: { required: true }
    kinds:
      decision: { name: "adr-{date:compact}-{slug}" }
      fact:     { name: "fact-{date:compact}-{slug}" }
  design:
    folders: [designs]
    kinds:
      design: { name: "{date}-{slug}" }

skeletons: _skeletons          # one file per kind, or per type when the kind has none

index:
  file: index.md
  groups: folder               # one "## <folder>" group per type folder
  sort: path

log:
  mode: dir                    # dir: one file per entry | file: one append-only file
  path: log
  operations: [init, ingest, knowledge, move, lint]

frozen:
  - path: archive
    checked: false             # its files are not pages; only "link leaves the base" applies
  - path: log
base_branch: main

citations:
  frontmatter: sources
  code: true

orphans: {}
```

What differs for Brain:

```yaml
root: .
locale: fr
title: filename
links: wikilink
ignore: [_meta, 90-templates, .claude, .obsidian, CLAUDE.md, AGENTS.md]
unchecked_frontmatter: [00-inbox]

types:
  wiki:
    folders: [20-wiki, 50-missions]
    frontmatter:
      status:   { required: true, values: [draft, done] }
      kind:     { required: true, values: [summary, concept, entity, comparison, answer] }
      sources:  { required: true, format: list, non_empty: true }
      reviewed: { required: true, format: date, empty_allowed: true }
    kinds:
      concept: { folder: 20-wiki/concepts, skeleton: wiki-concept, name: "{title}", set: { kind: concept } }
      summary: { folder: 20-wiki/resumes,  skeleton: wiki-resume,  name: "Résumé - {title}", set: { kind: summary } }
      entity:
        folder: 20-wiki/entites
        skeleton: wiki-entite
        name: "{title}"
        set: { kind: entity }
        frontmatter:
          entity_kind: { required: true, values: [person, tool, org] }
  meta: {}                     # no folders: index.md and log.md, at the vault root
  # … source, transcript, veille, experiment, module, post, note, mission

stray: false

index:
  file: index.md
  check: false                 # Brain has no index check today; project 3 decides
  include: [20-wiki]
  other_links: allow           # the index also links mission notes and .base files
  groups:
    20-wiki/concepts: "Concepts et comparaisons (`20-wiki/concepts/`)"
    20-wiki/resumes:  "Résumés de sources (`20-wiki/resumes/`)"

log:
  mode: file
  path: log.md
  operations: [ingest, ingest-rien, query, lint, move, init, mission]
  entry:
    labels: { created: "créé", modified: "modifié", note: "note" }
    separator: " : "           # "- créé : …"
    one_line: true             # items joined by ", " on one line
    as_links: true             # pages written as links, in the base's link style

frozen:
  - path: 10-raw
    except: [10-raw/zotero, 10-raw/AGENTS.md]
  - path: _meta/lint           # new reports only
append_only: [_meta/manifest.jsonl]

template_markers: ["{{", "<%"]

orphans:
  scope: [20-wiki, 50-missions]
  ignore_links_from: [index.md, log.md]
  ignore_links_from_types: [mission]
  exempt_types: [mission]

manifest:
  file: _meta/manifest.jsonl
  pipeline: ingest-v2
  statuses: [integrated, no_material, error, superseded]

protected:                     # paths the agent never writes (guard)
  - CLAUDE.md
  - AGENTS.md
  - .claude
  - _meta
  - 10-raw/zotero
  - wikai.yaml
```

### Meaning of each block

- **`root`, `ignore`** — `wikai` reads markdown files under `root`, minus `ignore` and `.git`. Every such file is a **page** and receives every check, with these exceptions: files of a frozen path marked `checked: false`; log entries in `dir` mode; skeletons (only their date and time markers are checked, see `template_markers`); the index and the log file when they carry no frontmatter (their links are still checked; with a frontmatter they are pages, as in Brain where they are `type: meta`). The page count printed by `check` is the number of pages.
- **`links`** — `markdown`: `[text](relative/path.md#anchor)`, resolved from the page's folder; the target must be a file tracked by git (or staged), with the exact case; the anchor must match a heading (lowercase, letters and digits kept, spaces to `-`, repeated headings take `-1`, `-2`). `wikilink`: `[[Name]]`, `[[Name#^block]]`, `[[Name|text]]`, resolved by file name or path suffix without case, in NFC, shortest path wins; a `^block` anchor must exist in the target. Links inside code fences and code spans are ignored in both styles. Wikilinks in frontmatter count.
- **`title`** — what names a page for duplicate detection and for index lines.
- **`frontmatter`** — per key: `required`, `format` (`date` = `YYYY-MM-DD`, `list`, text by default), `values` (allowed values), `non_empty`, `empty_allowed`. The top-level block applies to every page; a type adds its own, and so does each kind whose `set` values the page carries. For a key that several layers name, the rule of the last layer replaces the earlier ones whole, in the order top level, type, kinds by name: a type can relax a top-level rule. Frontmatter is parsed as YAML; invalid YAML is a finding.
- **`types`** — a page's `type` key must name a declared type. When the type has `folders`, the page must sit under one of them; a type without `folders` may sit anywhere. Several types may share a folder.
- **`stray`** (default `true`) — a markdown file under `root` that sits in no type folder and no frozen path, and is not the index, the log or a skeleton, is a finding.
- **`kinds`** — creation recipes of a type, used by `wikai new page <kind>`: target `folder` (default: the type's first folder), file `name` (default `{slug}`), `skeleton`, and `set` (frontmatter values to write, such as `kind: concept`). A type without `kinds` has one kind named after it. Name variables: `{title}`, `{slug}`, `{date}`, `{date:compact}`. A kind may add `frontmatter` rules: they apply to every page of the type whose frontmatter holds the kind's `set` values. This is how `entity_kind` is required on `kind: entity` pages only, without a condition syntax.
- **`index`** — `check` requires each page of `include` (default: every type folder) to be linked once. `other_links: forbid` (default) makes any index link that resolves to something else a finding; `allow` accepts it when the target exists. `check: false` turns the index check off; `new` and `move` still maintain the lines. `new page` inserts the line in the page's group, in `sort` order; `move` rewrites it. The line is `- <link> — <summary>`, the link written in the base's link style.
- **`log`** — `new log` writes an entry `## [YYYY-MM-DD] <operation> | <title>` followed by `- created:`, `- modified:`, `- note:` lines (one line per item, pages as plain paths). `entry` changes the shape: `labels`, `separator` (default `": "`), `one_line`, `as_links`. `dir`: file `<date>-<operation>-<slug>.md`, refused if it exists. `file`: appended to the end of the file. An operation outside `operations` is refused.
- **`frozen`, `append_only`, `base_branch`** — a frozen file may be added, never modified, renamed or deleted. An append-only file may only grow at its end. The reference is the merge-base of `HEAD` and `origin/<base_branch>`; `HEAD` when the repository has no remote; `--base <ref>` overrides.
- **`citations`** — parses `(source: X)` outside code. Each cited `X` must be in the frontmatter list, and each list entry must be cited. With `code: true`, a citation shaped `<repo>@<sha>:<path>` must be `<repo>@<sha7>:<path>#L<start>[-L<end>]` with a valid range, and matches the list entry without its range. A citation written as a markdown link matches by its target.
- **`orphans`** — a page in `scope` (default: every type folder) with no link from another page is a finding. `ignore_links_from` and `ignore_links_from_types` do not count as inbound; `exempt_types` are never orphans. The index never counts.
- **`template_markers`** — these strings left in a page are a finding; in a frozen path, only the frontmatter is searched. The `templates` check also reads every skeleton: a `{{date:FMT}}` or `{{time:FMT}}` that `new page` would refuse is a finding on the skeleton's path, so CI finds it before the first page of that kind. This skeleton check runs even when `template_markers` is empty. Skeletons still do not count as pages.
- **`unchecked_frontmatter`** — folders whose pages skip the frontmatter and type rules; their links are still checked.
- **`manifest`** — see `wikai manifest`.
- **`protected`** — see `wikai guard`.
- **`locale`** — language of the texts `wikai` writes into the base: the empty-section text (`Not documented.` / `Non renseigné.`), the empty-group text (`No page.` / `Aucune page.`), and the title and note of a `move` log entry (`Page X renamed Y`, `renamed from X` / `Page X renommée Y`, `renommé depuis X`).

## Commands

Exit codes of every command: 0 success, 1 refused or findings, 2 usage or environment error.

### `wikai check [--base <ref>] [--json]`

Runs every mechanism the config turns on. Checks: `stray`, `links`, `frontmatter`, `index`, `duplicates` (two pages share a title or an alias, without case), `orphans`, `sources`, `templates`, `frozen`.

Text output, one line per finding, then a summary:

```
orphans: knowledge/decisions/fact-20260101-example.md: no other page links to this page
FAIL: 1 finding(s)
```

or `OK: 148 pages`. `--json` prints the array `[{"check", "path", "message"}]`, the shape of the homes' `expected.json`.

### `wikai new page <kind> <title> --summary <text> [--slug S] [--date D] [--set key=value]...`

1. Refuses when the file exists, when the new page's title (its file name under `title: filename`) is the title or an alias of another page (it names that page), or, with wikilink links, when another file has the same name.
2. Copies the kind's skeleton and fills its markers (see below). Writes the `set` values of the kind. Fills required `date` keys left empty, except those marked `empty_allowed`. An empty section receives the empty-section text.
3. Adds the index line.
4. Prints the created path and the next steps (fill the sections, link the page, write the log entry).

Skeleton markers follow the Obsidian core Templates syntax. A marker is `{{name}}`; the name ends at the first `}}` and holds no `{`, `}` or line break. The skeleton is read once, from start to end: a filled value is never read again, so a `--summary` or `--set` value that holds `{{date}}` stays as written.

- `{{title}}`, `{{slug}}`, `{{summary}}`, and `{{key}}` for each `--set key=value`. `{{title}}` is the title passed to `new page`, not the file name as in Obsidian: a kind named `Résumé - {title}` gets the heading `# Talk`, not `# Résumé - Talk`.
- `{{date}}` is `{{date:YYYY-MM-DD}}`; `{{time}}` is `{{time:HH:mm}}`. These defaults are fixed. `{{date:}}` and `{{time:}}`, with nothing after the colon, give the same defaults.
- `{{date:FMT}}` and `{{time:FMT}}` are the same: they format the page's day with the Moment.js format `FMT`. The time is now, in local time, or 00:00 of the `--date` day.
- A `--set` key equal to `title`, `date`, `time`, `slug` or `summary`, or starting with `date:` or `time:`, is a usage error (exit 2).

Tokens of `FMT`, longest match first:

| Token | Output | Token | Output |
|---|---|---|---|
| `YYYY` `YY` | year, 4 and 2 digits | `H` `HH` | hour 0–23 |
| `GGGG` | ISO week year | `h` `hh` | hour 1–12 |
| `M` `MM` | month number | `m` `mm` | minute |
| `MMM` `MMMM` | month name, short and long | `s` `ss` | second |
| `D` `DD` | day of month | `A` `a` | `AM`/`PM`, `am`/`pm` |
| `Do` | day of month, ordinal | `W` `WW` | ISO week number |
| `dd` `ddd` `dddd` | weekday name, minimal, short, long | `Hmm` `Hmmss` | `H` then `mm`, then `ss` |
| | | `hmm` `hmmss` | `h` then `mm`, then `ss` |

Two-letter forms are zero-padded. Names and ordinals follow `locale`: `en` gives `January`, `Jan`, `Sunday`, `Sun`, `Su`, `1st`; `fr` gives `janvier`, `janv.`, `dimanche`, `dim.`, `di`, `1er` (then `2`, `3`…), as Moment's `fr` locale. `A` and `a` are the same in both locales. Text inside `[…]` and a character after `\` are written as they are. Any other Moment token (`Q`, `DDD`, `w`, `X`, `L`, `Z`…) is a usage error (exit 2) that names the skeleton and the token. Any other character, such as `-`, `/`, `:` or the `T` of `YYYY-MM-DDTHH:mm`, is written as it is.

A `{{…}}` with any other name stays in the page for the agent; `template_markers` reports it until it is replaced.

### `wikai new log <operation> <title> [--slug S] [--created P]... [--modified P]... [--note T]... [--date D]`

Writes the entry described under `log`. Needs at least one `--created`, `--modified` or `--note`.

### `wikai move <old> <new> [--no-log] [--note T]... [--date D]`

Runs `git mv`, rewrites every link to the page in every page, in the index and in the log unless the log is frozen or append-only (markdown: the relative path; wikilink: the name, when the file name changes, keeping `|text` and `#anchor`), re-sorts the index group, and writes a `move` log entry. A move to a frozen path is an archive: the index line is removed. A move to the folder of another type is refused, and so are a move that would change a link in a checked frozen page and, when the index is checked, a move into a folder that has no index group, and a move to a folder spelled in another case than the folder on disk. Links in frozen `checked: false` files are left as they are.

### `wikai manifest hash|status|append <note> [--status S] [--pages P,...] [--note T]`

Same behaviour and same record as `manifest.py`: a JSON line `{ts, raw, sha256, status, pipeline, pages, note}`, appended under a file lock. The hash is the SHA-256 of the normalised text: BOM removed, line ends to LF, frontmatter removed, trailing `^block` ids removed, whitespace runs merged to one space, trimmed. `status` prints `new`, `unchanged <status> <path>` or `modified`.

### `wikai guard hook`

A Claude Code `PreToolUse` hook for `Edit`, `Write` and `NotebookEdit`: reads the event on stdin, exits 2 with a message on stderr to refuse a write to a `protected` path, to an existing file of a `frozen` path, or to an `append_only` file unless the new content starts with the current content. Exits 0 otherwise, and for a path outside the boundary.

The hook applies every base that holds the target: each `wikai.yaml` above the target (its resolved path and its written path) and the base above the Claude Code project folder. It refuses when any of them refuses; an invalid `wikai.yaml` on the path refuses too. A nested `wikai.yaml` therefore never relaxes the rules of the base around it, and one hook installed for every project guards every base.

When a path matches several rules, `append_only` wins, then `frozen`, then `protected`. So in Brain the agent may append to `_meta/manifest.jsonl` and add a report to `_meta/lint/`, and nothing else under `_meta`. A path listed in a frozen entry's `except` follows no frozen rule; listing it under `protected` (`10-raw/zotero`) keeps the agent out while the human and their tools write it freely.

### `wikai guard commit (--staged | <base> <head>) [--agent]`

For a pre-commit hook or CI: refuses a change that modifies, renames or deletes a frozen file, or rewrites existing lines of an append-only file. With `--agent`, also refuses any change to a `protected` path, with the same precedence as the hook. The hook always applies `protected`: it only ever sees the agent.

### `wikai init [dir] --preset home-repo|obsidian-vault`

Writes `wikai.yaml`, the type folders, one skeleton per kind, an empty index and the log location. The date and time markers of the files other than skeletons are filled with today, as `new page` fills them. Refuses to overwrite an existing `wikai.yaml`. It writes no agent contract and no procedure: those come with the skills of project 4.

## Structure of the code

```
cmd/wikai/            entry point
internal/cli/         cobra commands, output, exit codes
internal/config/      wikai.yaml: load, validate, defaults
internal/wiki/        load a base: pages, frontmatter, links, headings, titles
internal/check/       one file per check; each takes the loaded base and returns findings
internal/write/       new page, new log, move, index edits
internal/manifest/
internal/guard/
internal/gitx/        the git calls
internal/preset/      embedded preset files for init
```

Each check is a function `(Base, Config) -> []Finding`, tested alone.

## Tests and acceptance

- The 29 fixtures of `am-home/knowledge/_ops/fixtures` (each a small base, a `base/` state for frozen paths, and an `expected.json`) are ported, renamed to fictitious names, and run as a table test: `wikai check --json` must return each `expected.json`. The test builds a temporary git repository from each `base/` folder. In the ported files the check names `archive` and `log` become `frozen`, and messages name "the base", not a home. dev-home's copy of the scripts is diffed against am-home's before porting (known differences: no code citations, `origin/master` tried before `origin/main`, French texts).
- Equivalent fixtures are written for the wikilink style and for each Brain-only mechanism (template markers, orphan exclusions, frozen exceptions, append-only, manifest).
- Manifest hashes are checked against values produced by `manifest.py` on synthetic notes.
- `new`, `move` and `guard` are tested on temporary git repositories.
- Acceptance: with a `wikai.yaml` written for each of the three bases, kept outside the bases and passed by `--config` and `--dir`, `wikai check` reports zero findings on all three, as the current scripts do today (Brain 129 notes, am-home 148 pages, dev-home 15 pages).
- `govulncheck ./...` reports nothing.
- A tagged release installs with `brew install --cask PillowPillow/tap/wikai` and `wikai --version` prints the tag.

## Known differences from the current scripts

- Frontmatter goes through a YAML parser instead of the two hand-written ones. A page the old parsers accepted may be invalid YAML; the acceptance run shows any such page, and project 3 decides what to do with it.
- The frozen check uses one rule for all bases (merge-base with `origin/<base_branch>`). Brain's script compared the working tree to `HEAD` by default; the new rule also catches a change committed locally and not pushed.
- `move` accepts an archive move from any type, where `new.py` allowed it from `designs/` only.
- Skeletons use `{{title}}`, `{{date}}`, `{{time}}`, `{{date:FMT}}`, `{{time:FMT}}`, `{{slug}}`, `{{summary}}` (Obsidian core Templates; Templater `<% %>` is not run). The current ones (`# <placeholder>` and empty `created:` in the homes, `{{AAAA-MM-JJ}}` in Brain) are rewritten in project 3; `wikai` does not read the old forms.
- `new.py page repo --origin <url>` derived the title and the file name from the git origin. Here a repo page is created with `--set origin=<url>` and an explicit title; the origin-aware command belongs to project 2.
- `base_branch` is one name per base; dev-home sets `master`.
- Code citations are on per base; dev-home has them off today. Project 3 decides whether to turn them on.
- CLI messages are English for Brain too.

## Left to the builder

- Exact wording of findings, where a ported fixture does not fix it.
- The contents of the two presets, beyond matching the examples above.
- Flag spelling and help text.
- Internal types and package boundaries below the level shown.
