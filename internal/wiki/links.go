package wiki

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type Link struct {
	Line      int
	Target    string
	Anchor    string
	HasAnchor bool
	External  bool
	Unparsed  bool
}

var (
	markdownLinkRe = regexp.MustCompile(`!?\[[^\]]*\]\(\s*(<[^>]*>|[^)\s]*)(?:\s+(?:"[^"]*"|'[^']*'))?\s*\)`)
	schemeRe       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	wikilinkRe     = regexp.MustCompile(`\[\[([^\[\]|#]*)(?:#([^\[\]|]*))?(?:\|[^\[\]]*)?\]\]`)
)

type linkMatch struct {
	link       Link
	start, end int
	angle      bool
}

func Links(style, text string) []Link {
	var out []Link
	for _, m := range scanLinks(style, text) {
		out = append(out, m.link)
	}
	return out
}

func RewriteLinks(style, text string, fn func(Link) (target string, ok bool)) string {
	var b strings.Builder
	last := 0
	for _, m := range scanLinks(style, text) {
		if m.link.Unparsed {
			continue
		}
		target, ok := fn(m.link)
		if !ok {
			continue
		}
		if style == "markdown" {
			target = escapeMarkdownPath(target, m.angle)
		}
		b.WriteString(text[last:m.start])
		b.WriteString(target)
		last = m.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func scanLinks(style, text string) []linkMatch {
	switch style {
	case "markdown":
		return markdownLinks(text)
	case "wikilink":
		return wikilinks(text)
	}
	return nil
}

func markdownLinks(text string) []linkMatch {
	var out []linkMatch
	for _, l := range codeLines(splitLines(text)) {
		covered := map[int]bool{}
		ends := map[int]bool{}
		for _, m := range markdownLinkRe.FindAllStringSubmatchIndex(l.text, -1) {
			for i := m[0]; i < m[1]; i++ {
				covered[i] = true
			}
			ends[m[1]] = true
			out = append(out, markdownLink(l, m[2], m[3]))
		}
		for i := 0; ; i++ {
			j := strings.Index(l.text[i:], "](")
			if j < 0 {
				break
			}
			i += j
			if !covered[i] && !ends[i] {
				out = append(out, linkMatch{link: Link{Line: l.number, Unparsed: true}})
			}
		}
	}
	return out
}

func markdownLink(l textLine, start, end int) linkMatch {
	raw := l.text[start:end]
	angle := len(raw) >= 2 && raw[0] == '<' && raw[len(raw)-1] == '>'
	if angle {
		start, end, raw = start+1, end-1, raw[1:len(raw)-1]
	}
	m := linkMatch{link: Link{Line: l.number}, start: l.start + start, end: l.start + end, angle: angle}
	if raw == "" || schemeRe.MatchString(raw) {
		m.link.Target, m.link.External = raw, true
		return m
	}
	path, anchor, hasAnchor := strings.Cut(raw, "#")
	path, _, _ = strings.Cut(path, "?")
	m.end = m.start + len(path)
	m.link.Target = percentDecode(path)
	if hasAnchor {
		m.link.Anchor, m.link.HasAnchor = percentDecode(anchor), true
	}
	return m
}

func wikilinks(text string) []linkMatch {
	lines := splitLines(text)
	var scanned []textLine
	if b, ok := findFrontmatter(text); ok {
		scanned = append(scanned, lines[:b.closingLine]...)
		lines = lines[b.closingLine:]
	}
	scanned = append(scanned, codeLines(lines)...)
	var out []linkMatch
	for _, l := range scanned {
		for _, m := range wikilinkRe.FindAllStringSubmatchIndex(l.text, -1) {
			out = append(out, wikilink(l, m))
		}
	}
	return out
}

func wikilink(l textLine, m []int) linkMatch {
	name := l.text[m[2]:m[3]]
	trimmed := strings.TrimLeftFunc(name, unicode.IsSpace)
	start := l.start + m[2] + len(name) - len(trimmed)
	target := strings.TrimSpace(strings.TrimSuffix(strings.TrimRightFunc(trimmed, unicode.IsSpace), `\`))
	link := Link{Line: l.number, Target: target}
	if m[4] >= 0 {
		link.Anchor, link.HasAnchor = l.text[m[4]:m[5]], true
		if m[5] < len(l.text) && l.text[m[5]] == '|' {
			link.Anchor = strings.TrimSuffix(link.Anchor, `\`)
		}
	}
	return linkMatch{link: link, start: start, end: start + len(target)}
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func unhex(c byte) byte {
	switch {
	case c <= '9':
		return c - '0'
	case c <= 'F':
		return c - 'A' + 10
	}
	return c - 'a' + 10
}

// Invalid escapes stay literal and "+" is not a space, as with Python's urllib.parse.unquote.
func percentDecode(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			b.WriteByte(unhex(s[i+1])<<4 | unhex(s[i+2]))
			i += 2
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func escapeMarkdownPath(path string, angle bool) string {
	var b strings.Builder
	for _, r := range path {
		escape := strings.ContainsRune("%#?<>", r) || !angle && (r == '(' || r == ')' || unicode.IsSpace(r))
		if !escape {
			b.WriteRune(r)
			continue
		}
		for _, c := range []byte(string(r)) {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
