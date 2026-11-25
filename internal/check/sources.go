package check

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
)

var (
	codeShape  = regexp.MustCompile(`^[A-Za-z0-9._-]+@[0-9A-Fa-f]+:`)
	codeSource = regexp.MustCompile(`^[A-Za-z0-9._-]+@[0-9a-f]{7}:[^#\s]+(?:#L(\d+)(?:-L(\d+))?)?$`)
	lineRange  = regexp.MustCompile(`#L\d+(?:-L\d+)?$`)
)

func Sources(b *wiki.Base) []wiki.Finding {
	c := b.Cfg.Citations
	if c == nil {
		return nil
	}
	var findings []wiki.Finding
	for _, p := range b.Pages {
		if p.Frozen {
			continue
		}
		list, ok := p.Meta[c.Frontmatter].([]string)
		if !ok {
			continue
		}
		add := func(msg string) {
			findings = append(findings, wiki.Finding{Check: "sources", Path: p.Path, Message: msg})
		}
		cited := map[string]bool{}
		for _, ct := range wiki.Citations(p.Text) {
			where := fmt.Sprintf("%s:%d:", p.Path, ct.Line)
			if ct.Unparsed {
				add(where + " unparsed citation")
				continue
			}
			key := ct.Key
			if c.Code && codeShape.MatchString(key) {
				m := codeSource.FindStringSubmatch(key)
				switch {
				case m == nil:
					add(where + " malformed code citation: " + key + " (expected <repo>@<sha7>:<path>#L<start>-L<end>)")
				case m[1] == "":
					add(where + " code citation without a line range: " + key)
				case lineNumber(m[1]) == 0:
					add(where + " line range starts at 0: " + key)
				case m[2] != "" && lineNumber(m[2]) < lineNumber(m[1]):
					add(where + " reversed line range: " + key)
				}
				if m != nil {
					key = lineRange.ReplaceAllString(key, "")
				}
			}
			cited[key] = true
			if !slices.Contains(list, key) {
				add(where + " cited but not in sources: " + key)
			}
		}
		for _, entry := range list {
			if !cited[entry] {
				add("in sources but never cited: " + entry)
			}
		}
	}
	return findings
}

func lineNumber(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return n
}
