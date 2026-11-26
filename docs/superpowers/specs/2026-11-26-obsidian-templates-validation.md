# wikai v0.2.0 date markers

*wikai · Obsidian core Templates markers in skeletons · 26 Nov 2026*

These statements replace a read of the spec change and the plan. Answer line by line; I read your answers from the session. The spec is aligned on this sheet, never the other way round.

## Contexte

wikai v0.2.0 lets a skeleton use the date and time markers of Obsidian's core Templates plugin, such as {{time}} and {{date:dddd D MMMM YYYY}}. The change touches page creation (wikai new page), the templates check of wikai check, and wikai init. The current release is v0.1.1; 'today' on this sheet means v0.1.1. A yes fixes the behaviour that the branch implements and that v0.2.0 ships; a no reopens the point before any code is written.

## Glossaire

- **base** : One knowledge base that wikai manages: a folder with a wikai.yaml file, its pages, an index and a log.
- **skeleton** : The markdown file that wikai new page copies to create a page of one kind; it lives in the base's skeletons folder.
- **marker** : A {{name}} placeholder in a skeleton that wikai new page replaces with a value, such as {{title}} or {{date}}.
- **Moment token** : A letter group of the Moment.js date format language that Obsidian uses, such as YYYY for the year or dddd for the weekday name; case matters (M is the month, m the minute).
- **FMT** : The format part of a {{date:FMT}} or {{time:FMT}} marker, written in Moment tokens, for example dddd D MMMM YYYY.
- **locale** : The wikai.yaml key (en or fr) that sets the language of the text wikai writes into pages.
- **templates check** : The wikai check rule that reports a page still holding a string listed under template_markers in wikai.yaml, such as {{.
- **preset** : A starter base that wikai init writes: wikai.yaml, skeletons, an index file and a log file. Two exist: home-repo and obsidian-vault.
- **the agent** : The LLM coding agent that runs wikai and then edits the page it created.
- **exit 2** : wikai's exit code for a usage or configuration error; a skeleton with a bad format counts as a configuration error.

Réponse vide = ligne non traitée. La fiche gagne sur le document qu'elle valide.

## A — Your decisions

I only check that my transcription matches what you said.

- **A1.** Skeletons may use {{time}}, {{date:FMT}} and {{time:FMT}}, as in Obsidian's core Templates plugin; markers of the Templater plugin, which run JavaScript, are out of scope. *(session brief)*
  - **Réponse : Faithful**
- **A2.** wikai formats dates with the Go standard library and a hand-written token map, with no new dependency. *(session brief)*
  - **Réponse : Faithful**
- **A3a.** A marker whose name wikai does not know, such as {{x}}, stays in the page as written, for the agent to replace. *(session brief)*
  - **Réponse : Faithful**
- **A3b.** A {{ that never closes with }} stays in the page as written. *(session brief)*
  - **Réponse : Faithful**
- **A3c.** Every value wikai puts into a skeleton (title, summary, --set values) must hold no line break. *(session brief)*
  - **Réponse : Faithful**
- **A3d.** wikai does not scan a filled value again: a summary that holds {{date}} stays {{date}} in the page. *(session brief)*
  - **Réponse : Faithful**
- **A4.** When the base's locale is fr, month and weekday names are French, taken from a table in wikai. *(session brief)*
  - **Réponse : Faithful**
- **A5.** Supported Moment tokens: YYYY YY M MM MMM MMMM D DD Do dd ddd dddd H HH h hh m mm s ss A a W WW GGGG; W, WW and GGGG follow ISO 8601 weeks. *(decision of 26 Nov on the token list)*
  - **Réponse : Faithful**
- **A6.** A Moment token outside the supported list, such as Q (quarter), stops wikai new page with exit 2. *(project ledger, templating decision)*
  - **Réponse : Faithful**
- **A7a.** --set refuses with exit 2 the keys title, date, time, slug and summary, and any key that starts with date: or time:. *(decision of 26 Nov on --set keys)*
  - **Réponse : Faithful**
- **A7b.** This breaks v0.1.1 calls: --set time=… filled {{time}} and now fails; --set title=… (and date, slug, summary) had no effect and now fails. *(decision of 26 Nov on --set keys)*
  - **Réponse : Faithful**
- **A8.** {{date}} always gives YYYY-MM-DD and {{time}} always gives HH:mm; no wikai.yaml key changes them. *(decision of 26 Nov on default formats)*
  - **Réponse : Faithful**
- **A9.** With --date, every time token prints midnight, so {{time}} gives 00:00 on any machine. *(decision of 26 Nov on --date)*
  - **Réponse : Faithful**
- **A10.** wikai check reads every skeleton and reports a date or time format that wikai new page would refuse. *(decision of 26 Nov on check)*
  - **Réponse : Faithful**
- **A11a.** The work ships from one branch and one PR; Claude does not merge it and does not tag it. *(session brief)*
  - **Réponse : Faithful**
- **A11b.** The human tags v0.2.0, and only then does the move of the am-home repository to wikai start. *(session brief)*
  - **Réponse : Faithful**

## B — My defaults

The real points to decide. Each line gives the retained choice, then the alternative.

### Marker syntax

- **B1.** A marker's name runs from {{ to the first }} and holds no {, } or line break; {{a{b}} is therefore not a marker and stays as written. *(spec, wikai new page)* — _Never discussed_
  - Conséquence : In {{{title}}}, the inner {{title}} is filled and the outer braces stay, as in v0.1.1.
  - Sinon : A marker's name is everything up to the first }}, including { and }.
  - **Réponse : Retained choice**

### Format tokens

- **B2.** A character of FMT that is not a Moment token prints as written, as Moment does: the T in {{date:YYYY-MM-DDTHH:mm}} prints T. *(spec, wikai new page)* — _Never discussed_
  - Sinon : Any letter of FMT that is not a supported token stops wikai new page with exit 2, so a literal letter must sit inside [ ].
  - **Réponse : Retained choice**
- **B2b.** wikai reads FMT exactly as Moment does, longest token first: YYY reads as the token YY followed by the letter Y, and a lone Y prints Y. *(spec, wikai new page)* — _Never discussed_
  - Sinon : A run of Y that is not YYYY or YY stops wikai new page with exit 2.
  - **Réponse : Retained choice**
- **B3.** A character after a backslash in FMT prints as written, as in Moment; text inside [ ] does too. *(spec, wikai new page)* — _Never discussed_
  - Sinon : Only [ ] escapes; a backslash prints as a backslash.
  - **Réponse : Retained choice**
- **B4.** {{date:FMT}} and {{time:FMT}} accept the same tokens and give the same text, as in Obsidian. *(spec, wikai new page)* — _Never discussed_
  - Sinon : {{date:FMT}} accepts only date tokens and {{time:FMT}} only time tokens; a mix is exit 2.
  - **Réponse : Retained choice**
- **B5.** When the locale is fr, A and a give AM/PM and am/pm, as Moment's fr locale does. *(spec, wikai new page)* — _Never discussed_
  - Sinon : When the locale is fr, A and a stop wikai new page with exit 2.
  - **Réponse : Retained choice**

### Day and time

- **B7.** Without --date, markers use the current time in the machine's local time zone. *(spec, wikai new page)* — _Never discussed_
  - Sinon : Without --date, markers use the current time in UTC.
  - **Réponse : Retained choice**

### Errors and checks

- **B8.** When a skeleton holds an unsupported token, wikai new page writes no page and no index entry, and its message names the skeleton and the token. *(spec, wikai new page)* — _Never discussed_
  - **Réponse : Agree**
- **B9.** wikai check reports a bad format in a skeleton under the templates check, with the skeleton's path. *(spec, template_markers)* — _Never discussed_
  - Sinon : wikai check reports it under a new check named skeletons.
  - **Réponse : Retained choice**
- **B9b.** Skeletons still do not count as pages: the OK: N pages line of wikai check keeps the same N. *(spec, root and ignore)* — _Never discussed_
  - Sinon : Skeletons count as pages in that line.
  - **Réponse : Retained choice**
- **B10.** Markers are filled everywhere in the skeleton, including the frontmatter and code blocks, as in v0.1.1. *(spec, wikai new page)* — _Never discussed_
  - Sinon : Markers inside code blocks stay as written.
  - **Réponse : Retained choice**

### wikai init

- **B11.** wikai init fills date and time markers in the preset's index and log files with the same rules as wikai new page. *(spec, wikai init)* — _Never discussed_
  - Conséquence : Those files hold only {{date}} today, so their output does not change.
  - Sinon : wikai init keeps replacing only {{date}} in those files, as in v0.1.1.
  - **Réponse : Retained choice**

### Release

- **B12.** The README gains one line that names the supported markers and points to the spec; no CHANGELOG file is added. *(plan, gates)* — _Never discussed_
  - Sinon : The README lists the full token table.
  - **Réponse : Retained choice**

## C — What you do not have to validate

Facts verified in code or in primary sources, and implementation detail. If one of them contradicts A or B, A or B wins. Today a skeleton is filled by one strings.NewReplacer call (internal/write/page.go:104-112); the built-in values come first, so --set title=X is already ignored and --set time=X fills {{time}}. The --date flag is parsed to 00:00 UTC (internal/cli/new.go:14-19) and formatted in UTC, so it prints the given day and 00:00 on any machine; without it the command uses the local clock. wikai check does not read skeletons today: they are not pages. Go's time.Format("YYYY-MM-DD") returns the string unchanged, so Moment formats need a token map; Go has no escape character, so wikai formats one token at a time. Moment's format grammar (src/lib/format/format.js, develop branch): one regex of tokens, [..] and backslash escapes, and a one-character fallback that prints any other character as written. Moment's fr locale: janvier…décembre, janv. févr. mars avr. mai juin juil. août sept. oct. nov. déc., dimanche…samedi, dim. lun. mar. mer. jeu. ven. sam., di lu ma me je ve sa, day ordinal 1er then the plain number, no French AM/PM text. English minimal weekdays: Su Mo Tu We Th Fr Sa. Implementation: a new package internal/skeleton with Fill, Check and Reserved, used by new page, check and init; the tokenizer ports Moment's regex. Eight red-green slices on branch feat/obsidian-templates; gates vet, gofmt, go test, govulncheck with the local database, shellcheck, goreleaser snapshot; read-only acceptance rerun on the three bases before the PR. From memory, unverified: Obsidian keeps its default date and time formats in .obsidian/templates.json. No such file was found on this machine.
