package guard

import (
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/config"
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

func Classify(cfg *config.Config, p string) Rule {
	if slices.ContainsFunc(config.Segments(p), func(s string) bool { return strings.EqualFold(s, ".git") }) {
		return Protected
	}
	for _, a := range cfg.AppendOnly {
		if config.SamePath(p, cfg.RootPath(a)) {
			return AppendOnly
		}
	}
	for _, f := range cfg.Frozen {
		if !config.Under(p, cfg.RootPath(f.Path)) {
			continue
		}
		if !slices.ContainsFunc(f.Except, func(e string) bool { return config.Under(p, cfg.RootPath(e)) }) {
			return Frozen
		}
	}
	for _, e := range cfg.Protected {
		if config.Under(p, e) {
			return Protected
		}
	}
	return None
}
