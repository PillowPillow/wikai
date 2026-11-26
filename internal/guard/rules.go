package guard

import (
	"path"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/config"
	"golang.org/x/text/unicode/norm"
)

type Rule int

const (
	None Rule = iota
	AppendOnly
	Frozen
	Protected
)

func (r Rule) String() string {
	return [...]string{"none", "append-only", "frozen", "protected"}[r]
}

// Rules match case-insensitively on every filesystem: on a case-insensitive one, agents.md is AGENTS.md.
func Classify(cfg *config.Config, p string) Rule {
	segs := split(p)
	if slices.ContainsFunc(segs, func(s string) bool { return strings.EqualFold(s, ".git") }) {
		return Protected
	}
	for _, a := range cfg.AppendOnly {
		if e := split(cfg.RootPath(a)); len(e) == len(segs) && hasPrefix(segs, e) {
			return AppendOnly
		}
	}
	for _, f := range cfg.Frozen {
		if !hasPrefix(segs, split(cfg.RootPath(f.Path))) {
			continue
		}
		if !slices.ContainsFunc(f.Except, func(e string) bool { return hasPrefix(segs, split(cfg.RootPath(e))) }) {
			return Frozen
		}
	}
	for _, e := range cfg.Protected {
		if hasPrefix(segs, split(e)) {
			return Protected
		}
	}
	return None
}

func split(p string) []string {
	p = norm.NFC.String(path.Clean(p))
	if p == "." {
		return nil
	}
	return strings.Split(p, "/")
}

func hasPrefix(segs, prefix []string) bool {
	return len(prefix) <= len(segs) && slices.EqualFunc(segs[:len(prefix)], prefix, strings.EqualFold)
}
