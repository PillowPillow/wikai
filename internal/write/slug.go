package write

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(norm.NFKD.String(s)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		} else if r < 0x80 {
			dash = true
		}
	}
	return b.String()
}
