package wiki

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type Line struct {
	Number int
	Text   string
}

type textLine struct {
	number int
	start  int
	text   string
}

var (
	fenceRe   = regexp.MustCompile("^\\s*(`{3,}|~{3,})")
	headingRe = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$`)
	titleRe   = regexp.MustCompile(`^# +(.*?)\s*#*\s*$`)
)

func splitLines(text string) []textLine {
	var lines []textLine
	for start, number := 0, 1; start < len(text); number++ {
		end, next := len(text), len(text)
		if i := strings.IndexByte(text[start:], '\n'); i >= 0 {
			end, next = start+i, start+i+1
		}
		lines = append(lines, textLine{number, start, strings.TrimSuffix(text[start:end], "\r")})
		start = next
	}
	return lines
}

func outsideFences(lines []textLine) []textLine {
	var out []textLine
	fence := ""
	for _, l := range lines {
		m := fenceRe.FindStringSubmatch(l.text)
		switch {
		case fence == "" && m != nil:
			fence = m[1]
		case fence == "":
			out = append(out, l)
		case m != nil && m[1][0] == fence[0] && len(m[1]) >= len(fence):
			fence = ""
		}
	}
	return out
}

func codeLines(lines []textLine) []textLine {
	out := outsideFences(lines)
	for i := range out {
		out[i].text = blankCodeSpans(out[i].text)
	}
	return out
}

func backtickRun(b []byte, i int) int {
	n := 0
	for i+n < len(b) && b[i+n] == '`' {
		n++
	}
	return n
}

// Spaces replace code spans byte for byte so match offsets stay valid in the original line.
func blankCodeSpans(s string) string {
	if !strings.Contains(s, "`") {
		return s
	}
	b := []byte(s)
	for i := 0; i < len(b); {
		if b[i] != '`' {
			i++
			continue
		}
		n := backtickRun(b, i)
		end := -1
		for j := i + n; j < len(b); {
			if b[j] != '`' {
				j++
				continue
			}
			m := backtickRun(b, j)
			if m == n {
				end = j + m
				break
			}
			j += m
		}
		if end < 0 {
			i += n
			continue
		}
		for k := i; k < end; k++ {
			b[k] = ' '
		}
		i = end
	}
	return string(b)
}

func OutsideCode(text string) []Line {
	var out []Line
	for _, l := range codeLines(splitLines(text)) {
		out = append(out, Line{l.number, l.text})
	}
	return out
}

func HeadingTitle(body string) (string, bool) {
	for _, l := range outsideFences(splitLines(body)) {
		if m := titleRe.FindStringSubmatch(l.text); m != nil {
			return m[1], true
		}
	}
	return "", false
}

func Headings(text string) []string {
	var out []string
	for _, l := range outsideFences(splitLines(text)) {
		if m := headingRe.FindStringSubmatch(l.text); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

func slugify(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Anchors(text string) map[string]bool {
	anchors := map[string]bool{}
	seen := map[string]int{}
	for _, h := range Headings(text) {
		slug := slugify(h)
		if n := seen[slug]; n > 0 {
			anchors[fmt.Sprintf("%s-%d", slug, n)] = true
		} else {
			anchors[slug] = true
		}
		seen[slug]++
	}
	return anchors
}

func HasBlockID(text, id string) bool {
	if id == "" {
		return false
	}
	for _, l := range splitLines(text) {
		if strings.HasSuffix(strings.TrimRightFunc(l.text, unicode.IsSpace), "^"+id) {
			return true
		}
	}
	return false
}
