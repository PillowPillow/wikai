package wiki

import "strings"

func FoldName(s string) string {
	return strings.ToLower(nfc(strings.TrimSpace(s)))
}

func Names(p *Page) []string {
	raw := []string{p.Title}
	switch a := p.Meta["aliases"].(type) {
	case string:
		raw = append(raw, a)
	case []string:
		raw = append(raw, a...)
	case []any:
		for _, v := range a {
			if s, ok := v.(string); ok {
				raw = append(raw, s)
			}
		}
	}
	var names []string
	for _, n := range raw {
		if n = FoldName(n); n != "" {
			names = append(names, n)
		}
	}
	return names
}

func EscapeTarget(path string) string { return escapeMarkdownPath(path, false) }
