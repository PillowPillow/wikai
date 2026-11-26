package config

import (
	"path"
	"slices"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Segments splits a slash path after NFC and path.Clean; "." gives no segment.
func Segments(p string) []string {
	p = norm.NFC.String(path.Clean(p))
	if p == "." {
		return nil
	}
	return strings.Split(p, "/")
}

// Under reports whether p is dir or inside it, segment by segment, folding case so a rule still matches on a case-insensitive filesystem.
func Under(p, dir string) bool {
	segs, prefix := Segments(p), Segments(dir)
	return len(prefix) <= len(segs) && slices.EqualFunc(segs[:len(prefix)], prefix, strings.EqualFold)
}

func UnderAny(p string, dirs []string) bool {
	return slices.ContainsFunc(dirs, func(d string) bool { return Under(p, d) })
}

func SamePath(a, b string) bool {
	return len(Segments(a)) == len(Segments(b)) && Under(a, b)
}
