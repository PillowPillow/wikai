# wikai

`wikai` checks, writes and guards LLM wikis: folders of markdown pages whose rules (frontmatter, links, index, log, frozen and protected paths) live in one `wikai.yaml` at the root of the base. It gives an agent and a CI job the same verdict, and gives the agent commands to create and move pages without breaking those rules.

## Install

```sh
brew install --cask PillowPillow/tap/wikai
```

Without Homebrew (darwin and linux, amd64 and arm64):

```sh
curl -fsSL https://raw.githubusercontent.com/PillowPillow/wikai/main/install.sh | sh
```

`WIKAI_VERSION=v0.1.0` pins a version and `WIKAI_INSTALL_DIR=~/bin` changes the destination (default `~/.local/bin`); set them on `sh`, not on `curl`.

## Commands

- `wikai check [--base <ref>] [--json]`: run every check the config turns on and exit 1 on findings.
- `wikai new page <kind> <title> --summary <text>`: create a page from its skeleton and add its index line; refused when another page has the same name.
- `wikai new log <operation> <title>`: write a log entry.
- `wikai move <old> <new>`: `git mv` a page and rewrite every link to it; refused when it would change a link in a checked frozen page.
- `wikai manifest hash|status|append <note>`: hash a note and read or append its manifest record.
- `wikai guard hook`: Claude Code `PreToolUse` hook that refuses writes to protected, frozen and append-only paths.
- `wikai guard commit (--staged | <base> <head>) [--agent]`: refuse a change that breaks a frozen or append-only file, for a pre-commit hook or CI.
- `wikai init [dir] --preset home-repo|obsidian-vault`: write a starter `wikai.yaml`, folders, skeletons, index and log location.

Exit codes: 0 success, 1 refused or findings, 2 usage or environment error.

## `wikai.yaml`

The file sits at the root of the base; commands find it by walking up from the current folder. Its folder is the boundary.

Path rule: `root` and `protected` are relative to the boundary; every other path in the file is relative to `root`. Paths that `wikai` prints or records are relative to the boundary. Path arguments on the command line are resolved from the current folder.

Example for a home repository:

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
    frontmatter:               # adds keys; a key named above gets this rule instead
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
protected: [wikai.yaml, .claude] # relative to the boundary, not to root
base_branch: main

citations:
  frontmatter: sources
  code: true

orphans: {}
```

Each block is described in [Meaning of each block](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block):

- [`version`, `root`, `locale`, `title`, `links`](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block)
- [`frontmatter`, `types`](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block)
- [`skeletons`, `index`, `log`](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block)
- [`frozen`, `base_branch`](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block)
- [`citations`, `orphans`](docs/superpowers/specs/2026-11-25-wikai-core-design.md#meaning-of-each-block)

The full format and the behaviour of every command are in the [design spec](docs/superpowers/specs/2026-11-25-wikai-core-design.md).

## Guard an agent

Refuse agent writes to protected paths with a Claude Code hook in `.claude/settings.json`, or in `~/.claude/settings.json` to guard every base from any project (the hook applies each base that holds the file):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write|NotebookEdit",
        "hooks": [{ "type": "command", "command": "wikai guard hook" }]
      }
    ]
  }
}
```

Refuse commits that break frozen or append-only files with a git pre-commit hook:

```sh
exec wikai guard commit --staged
```

## Licence

[GPL-3.0](LICENSE)
