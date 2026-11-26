# Obsidian core Templates markers — plan (v0.2.0)

Spec: `docs/superpowers/specs/2026-11-25-wikai-core-design.md`, `wikai new page` (skeleton markers), `template_markers`, `wikai init`.
Branch: `feat/obsidian-templates`, one PR. The human tags v0.2.0.

## Design

- New package `internal/skeleton`, one deep module:
  - `Fill(text string, values map[string]string, day time.Time, locale string) (string, error)`: one pass from start to end; a filled value is never read again.
  - `Check(text string) []string`: the unsupported tokens of every `{{date:FMT}}` and `{{time:FMT}}`.
  - `Reserved(key string) bool`: the `--set` keys that collide with a built-in value.
- The format tokenizer ports Moment's `formattingTokens` regex (moment `src/lib/format/format.js`, `develop`) and its `localFormattingTokens` (`L`, `l` forms). For each match: brackets or `\` give literal text; a supported token is formatted; any other Moment token is an error; the single-character fallback is literal. Numbers come from `time.Format` with the mapped Go layout, one token at a time; names, `Do` and `dd` come from a locale table (`en`, `fr`, strings from moment `src/locale/fr.js`).
- `write.NewPage` replaces its `strings.NewReplacer` with `skeleton.Fill`; `preset.Init` uses `Fill` with no values for files other than skeletons; `check.Templates` runs `Check` on every skeleton.
- `cli.parseDate` stays: 00:00 UTC formats as the same day and as `00:00`, and no supported token prints a zone.

## Slices (red-green, each one commit or more, gate green at the end of each)

1. **Tracer bullet** — `{{time}}` and `{{date:YYYY-MM-DD}}`, `{{date:DD/MM/YY HH:mm}}` filled by `wikai new page`; `--date` gives 00:00. Keep: unknown `{{x}}` stays, a literal `{{` stays, a `--summary`/`--set` value holding `{{date:YYYY}}` stays literal (`page_test.go:112` invariant).
2. **Every supported token** — `M D H h hh m s A a W WW GGGG`; `[…]` and `\` literals; `T` and other non-token characters literal; `Y` alone literal (Moment). Table test against values computed by hand, ISO week edge (2027-01-01 → week 53 of 2026).
3. **Names and ordinals** — `MMM MMMM dd ddd dddd Do` in `en` and `fr`.
4. **Unsupported tokens** — `Q`, `DDD`, `w`, `X`, `LL`, `Z`, `Mo` → exit 2 naming the skeleton and the token; no file written.
5. **Reserved `--set` keys** — `title date time slug summary date:… time:…` → exit 2.
6. **`check` reads skeletons** — a bad format in a skeleton is a `templates` finding on the skeleton path; page count unchanged; fixture test.
7. **`init`** — preset index and log filled through `Fill` (behaviour unchanged, test kept).
8. **Docs** — README one line on skeleton markers; ledger and handoff.

Blocking edges: 1 → 2 → 3, 1 → 4 → 6, 1 → 5, 1 → 7.

## Gates

`go vet ./...`, `gofmt -l .` empty, `go test ./...`, `govulncheck ./...` (local db), `shellcheck install.sh acceptance/*.sh`, `goreleaser release --snapshot --clean`. Acceptance rerun on the three bases (read-only) before the PR: same counts as at 414a9f0.
