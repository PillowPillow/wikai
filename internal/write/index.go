package write

import (
	"path"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func nfc(s string) string { return norm.NFC.String(s) }

func Indexed(b *wiki.Base, pagePath string) bool {
	_, _, ok := indexGroup(b, pagePath)
	return ok
}

func indexDir(b *wiki.Base, pagePath string) (string, bool) {
	idx := b.Cfg.Index
	if idx == nil {
		return "", false
	}
	pagePath = nfc(pagePath)
	if !slices.ContainsFunc(idx.Include, func(f string) bool { return config.Under(pagePath, nfc(b.Cfg.RootPath(f))) }) {
		return "", false
	}
	dir := path.Dir(pagePath)
	if b.Cfg.Root != "." {
		rel, found := strings.CutPrefix(dir, b.Cfg.Root)
		if !found || rel != "" && rel[0] != '/' {
			return "", false
		}
		dir = path.Clean(strings.TrimPrefix(rel, "/"))
	}
	return dir, true
}

func RequireGroup(b *wiki.Base, pagePath string) error {
	dir, included := indexDir(b, pagePath)
	if included && b.Cfg.Index.Check && !Indexed(b, pagePath) {
		return exit.Usage("no index group for %s/: add it to index.groups", dir)
	}
	return nil
}

func indexGroup(b *wiki.Base, pagePath string) (heading string, flat, ok bool) {
	idx := b.Cfg.Index
	if idx == nil {
		return "", false, false
	}
	dir, included := indexDir(b, pagePath)
	if !included {
		return "", false, false
	}
	switch {
	case idx.Groups.ByFolder:
		best := ""
		for _, t := range b.Cfg.Types {
			for _, f := range t.Folders {
				if f = nfc(path.Clean(f)); config.Under(dir, f) && len(f) > len(best) {
					best = f
				}
			}
		}
		return best, false, best != ""
	case idx.Groups.Headings != nil:
		best := ""
		for key := range idx.Groups.Headings {
			if k := nfc(path.Clean(key)); config.Under(dir, k) && len(k) > len(best) {
				best = key
			}
		}
		if best == "" {
			return "", false, false
		}
		return idx.Groups.Headings[best], false, true
	}
	return "", true, true
}

func LinkTo(b *wiki.Base, fromFile, pagePath, text string) string {
	if b.Cfg.Links == "wikilink" {
		name := strings.TrimSuffix(path.Base(pagePath), ".md")
		if p, r := b.Resolve(fromFile, wiki.Link{Target: name}); r == wiki.Resolved && p != nfc(pagePath) {
			name = wikiPath(b, nfc(pagePath))
		}
		return "[[" + name + "]]"
	}
	rel := relPath(path.Dir(fromFile), pagePath)
	text = strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(text)
	return "[" + text + "](" + wiki.EscapeTarget(rel) + ")"
}

func relPath(fromDir, target string) string {
	from := strings.Split(path.Clean(fromDir), "/")
	if from[0] == "." {
		from = nil
	}
	to := strings.Split(target, "/")
	i := 0
	for i < len(from) && i < len(to)-1 && from[i] == to[i] {
		i++
	}
	parts := slices.Repeat([]string{".."}, len(from)-i)
	return strings.Join(append(parts, to[i:]...), "/")
}

func entryTarget(style, line string) (string, bool) {
	if !strings.HasPrefix(line, "- ") {
		return "", false
	}
	for _, l := range wiki.Links(style, line) {
		if l.External || l.Unparsed || l.Target == "" {
			return "", false
		}
		return l.Target, true
	}
	return "", false
}

func sortKey(style, target string) string {
	if style == "wikilink" {
		return wiki.FoldName(target)
	}
	return nfc(target)
}

func sameTarget(b *wiki.Base, indexPath, target, pagePath string) bool {
	if b.Cfg.Links == "wikilink" {
		t := strings.TrimSuffix(wiki.FoldName(target), ".md")
		p := strings.ToLower(nfc(strings.TrimSuffix(pagePath, ".md")))
		return p == t || strings.HasSuffix(p, "/"+t)
	}
	return nfc(path.Join(path.Dir(indexPath), target)) == nfc(pagePath)
}

func rewriteIndex(b *wiki.Base, index, pagePath, newTarget string, change func(entries []string) []string) (string, error) {
	if !strings.Contains(index, "\r\n") {
		return rewriteIndexLF(b, index, pagePath, newTarget, change)
	}
	out, err := rewriteIndexLF(b, strings.ReplaceAll(index, "\r\n", "\n"), pagePath, newTarget, change)
	return strings.ReplaceAll(out, "\n", "\r\n"), err
}

func rewriteIndexLF(b *wiki.Base, index, pagePath, newTarget string, change func(entries []string) []string) (string, error) {
	heading, flat, ok := indexGroup(b, pagePath)
	if !ok {
		return index, nil
	}
	style := b.Cfg.Links
	lines := strings.Split(strings.TrimSuffix(index, "\n"), "\n")
	if index == "" {
		lines = nil
	}
	if flat {
		return rewriteFlat(b, lines, pagePath, newTarget, change), nil
	}
	start := slices.Index(lines, "## "+heading)
	if start < 0 {
		return "", exit.Usage("%s has no group \"## %s\"", b.IndexPath(), heading)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	empty := Locale(b.Cfg.Locale).EmptyGroup
	var other, entries []string
	for _, l := range lines[start+1 : end] {
		switch {
		case strings.TrimSpace(l) == "" || strings.TrimSpace(l) == empty:
		default:
			if _, isEntry := entryTarget(style, l); isEntry {
				entries = append(entries, l)
			} else {
				other = append(other, l)
			}
		}
	}
	body := append(other, sortEntries(style, change(entries))...)
	if len(body) == 0 {
		body = []string{empty}
	}
	out := append([]string{}, lines[:start+1]...)
	out = append(out, "")
	out = append(out, body...)
	if end < len(lines) {
		out = append(out, "")
	}
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n") + "\n", nil
}

func rewriteFlat(b *wiki.Base, lines []string, pagePath, newTarget string, change func([]string) []string) string {
	style := b.Cfg.Links
	var runs [][2]int
	for i, l := range lines {
		if _, ok := entryTarget(style, l); !ok {
			continue
		}
		if n := len(runs); n > 0 && runs[n-1][1] == i {
			runs[n-1][1] = i + 1
		} else {
			runs = append(runs, [2]int{i, i + 1})
		}
	}
	if len(runs) == 0 {
		out := slices.Clone(lines)
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, sortEntries(style, change(nil))...)
		return strings.Join(out, "\n") + "\n"
	}
	chosen := len(runs) - 1
	found := false
	for r, run := range runs {
		for _, l := range lines[run[0]:run[1]] {
			if t, _ := entryTarget(style, l); sameTarget(b, b.IndexPath(), t, pagePath) {
				chosen, found = r, true
			}
		}
		if found {
			break
		}
	}
	if !found && newTarget != "" {
		key := sortKey(style, newTarget)
		for r, run := range runs {
			first, _ := entryTarget(style, lines[run[0]])
			last, _ := entryTarget(style, lines[run[1]-1])
			if sortKey(style, first) <= key && key <= sortKey(style, last) {
				chosen = r
				break
			}
		}
	}
	run := runs[chosen]
	out := slices.Clone(lines[:run[0]])
	out = append(out, sortEntries(style, change(slices.Clone(lines[run[0]:run[1]])))...)
	out = append(out, lines[run[1]:]...)
	return strings.Join(out, "\n") + "\n"
}

func sortEntries(style string, entries []string) []string {
	out := slices.Clone(entries)
	key := func(l string) string {
		t, _ := entryTarget(style, l)
		return sortKey(style, t)
	}
	slices.SortStableFunc(out, func(a, b string) int { return strings.Compare(key(a), key(b)) })
	return out
}

func IndexAdd(b *wiki.Base, index, pagePath, title, summary string) (string, error) {
	line := "- " + LinkTo(b, b.IndexPath(), pagePath, title)
	if s := strings.Join(strings.Fields(summary), " "); s != "" {
		line += " — " + s
	}
	newTarget := relPath(path.Dir(b.IndexPath()), pagePath)
	if b.Cfg.Links == "wikilink" {
		newTarget = strings.TrimSuffix(path.Base(pagePath), ".md")
	}
	return rewriteIndex(b, index, pagePath, newTarget, func(entries []string) []string { return append(entries, line) })
}

func IndexRemove(b *wiki.Base, index, pagePath string) (string, error) {
	return rewriteIndex(b, index, pagePath, "", func(entries []string) []string {
		return slices.DeleteFunc(entries, func(l string) bool {
			t, _ := entryTarget(b.Cfg.Links, l)
			return sameTarget(b, b.IndexPath(), t, pagePath)
		})
	})
}

func IndexResort(b *wiki.Base, index, pagePath string) (string, error) {
	return rewriteIndex(b, index, pagePath, "", func(entries []string) []string { return entries })
}
