package wiki

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Citation struct {
	Line     int
	Key      string
	Unparsed bool
}

func Citations(text string) []Citation {
	var out []Citation
	for _, l := range codeLines(splitLines(text)) {
		s := l.text
		parsed := map[int]bool{}
		for i := 0; i < len(s); {
			j := strings.Index(s[i:], "(source")
			if j < 0 {
				break
			}
			i += j
			if key, end, ok := matchCitation(s, i); ok {
				out = append(out, Citation{Line: l.number, Key: key})
				parsed[i] = true
				i = end
				continue
			}
			i++
		}
		for i := 0; i < len(s); i++ {
			j := strings.Index(s[i:], "(source")
			if j < 0 {
				break
			}
			i += j
			if _, ok := citationOpener(s, i); ok && !parsed[i] {
				out = append(out, Citation{Line: l.number, Unparsed: true})
			}
		}
	}
	return out
}

func skipSpace(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i += size
	}
	return i
}

func citationOpener(s string, i int) (int, bool) {
	if !strings.HasPrefix(s[i:], "(source") {
		return 0, false
	}
	i = skipSpace(s, i+len("(source"))
	if i >= len(s) || s[i] != ':' {
		return 0, false
	}
	return i + 1, true
}

// The text form ends at the first ")" reached with parentheses balanced one level deep.
func matchCitation(s string, i int) (key string, end int, ok bool) {
	p, ok := citationOpener(s, i)
	if !ok {
		return "", 0, false
	}
	p = skipSpace(s, p)
	if p < len(s) && s[p] == '[' {
		return linkCitation(s, p)
	}
	for pos := p; ; {
		if q := skipSpace(s, pos); q < len(s) && s[q] == ')' {
			return strings.TrimSpace(s[p:pos]), q + 1, true
		}
		if pos >= len(s) {
			return "", 0, false
		}
		if s[pos] == '(' {
			k := strings.IndexAny(s[pos+1:], "()")
			if k < 0 || s[pos+1+k] == '(' {
				return "", 0, false
			}
			pos += k + 2
			continue
		}
		_, size := utf8.DecodeRuneInString(s[pos:])
		pos += size
	}
}

func linkCitation(s string, p int) (string, int, bool) {
	q := strings.IndexByte(s[p:], ']')
	if q < 0 || p+q+1 >= len(s) || s[p+q+1] != '(' {
		return "", 0, false
	}
	start := skipSpace(s, p+q+2)
	r := start
	for r < len(s) && s[r] != ')' {
		c, size := utf8.DecodeRuneInString(s[r:])
		if unicode.IsSpace(c) {
			break
		}
		r += size
	}
	target := s[start:r]
	for range 2 {
		r = skipSpace(s, r)
		if r >= len(s) || s[r] != ')' {
			return "", 0, false
		}
		r++
	}
	return target, r, true
}
