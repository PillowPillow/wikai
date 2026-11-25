package write

import (
	"path"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func nfc(s string) string { return norm.NFC.String(s) }

func inDir(p, dir string) bool {
	return dir == "." || p == dir || strings.HasPrefix(p, dir+"/")
}

// Reports whether the index holds a line for the page: the index is on, the page is in index.include and a group applies.
func Indexed(b *wiki.Base, pagePath string) bool {
	_, _, ok := indexGroup(b, pagePath)
	return ok
}

func indexGroup(b *wiki.Base, pagePath string) (heading string, flat, ok bool) {
	idx := b.Cfg.Index
	if idx == nil {
		return "", false, false
	}
	pagePath = nfc(pagePath)
	if !slices.ContainsFunc(idx.Include, func(f string) bool { return inDir(pagePath, nfc(b.Cfg.RootPath(f))) }) {
		return "", false, false
	}
	dir := path.Dir(pagePath)
	if b.Cfg.Root != "." {
		rel, found := strings.CutPrefix(dir, b.Cfg.Root)
		if !found || rel != "" && rel[0] != '/' {
			return "", false, false
		}
		dir = path.Clean(strings.TrimPrefix(rel, "/"))
	}
	switch {
	case idx.Groups.ByFolder:
		best := ""
		for _, t := range b.Cfg.Types {
			for _, f := range t.Folders {
				if f = nfc(path.Clean(f)); inDir(dir, f) && len(f) > len(best) {
					best = f
				}
			}
		}
		return best, false, best != ""
	case idx.Groups.Headings != nil:
		best := ""
		for key := range idx.Groups.Headings {
			if k := nfc(path.Clean(key)); inDir(dir, k) && len(k) > len(best) {
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
		return "[[" + strings.TrimSuffix(path.Base(pagePath), ".md") + "]]"
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

func rewriteIndex(b *wiki.Base, index, pagePath string, change func(entries []string) []string) (string, error) {
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
		return rewriteFlat(style, lines, change), nil
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

func rewriteFlat(style string, lines []string, change func([]string) []string) string {
	var entries []string
	last := -1
	for i, l := range lines {
		if _, ok := entryTarget(style, l); ok {
			entries = append(entries, l)
			last = i
		}
	}
	sorted := sortEntries(style, change(entries))
	var out []string
	next := 0
	for i, l := range lines {
		if _, ok := entryTarget(style, l); ok {
			if next < len(sorted) {
				out = append(out, sorted[next])
				next++
			}
		} else {
			out = append(out, l)
		}
		if i == last {
			out = append(out, sorted[next:]...)
			next = len(sorted)
		}
	}
	if last < 0 {
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, sorted...)
	}
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
	return rewriteIndex(b, index, pagePath, func(entries []string) []string { return append(entries, line) })
}

func IndexRemove(b *wiki.Base, index, pagePath string) (string, error) {
	return rewriteIndex(b, index, pagePath, func(entries []string) []string {
		return slices.DeleteFunc(entries, func(l string) bool {
			t, _ := entryTarget(b.Cfg.Links, l)
			return sameTarget(b, b.IndexPath(), t, pagePath)
		})
	})
}

func IndexResort(b *wiki.Base, index, pagePath string) (string, error) {
	return rewriteIndex(b, index, pagePath, func(entries []string) []string { return entries })
}
