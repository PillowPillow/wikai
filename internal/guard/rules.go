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

func Classify(cfg *config.Config, p string) Rule {
	p = norm.NFC.String(path.Clean(p))
	if slices.Contains(strings.Split(p, "/"), ".git") {
		return Protected
	}
	for _, a := range cfg.AppendOnly {
		if p == norm.NFC.String(cfg.RootPath(a)) {
			return AppendOnly
		}
	}
	for _, f := range cfg.Frozen {
		if !under(p, cfg.RootPath(f.Path)) {
			continue
		}
		if !slices.ContainsFunc(f.Except, func(e string) bool { return under(p, cfg.RootPath(e)) }) {
			return Frozen
		}
	}
	for _, e := range cfg.Protected {
		if under(p, path.Clean(e)) {
			return Protected
		}
	}
	return None
}

func under(p, dir string) bool {
	dir = norm.NFC.String(dir)
	return dir == "." || p == dir || strings.HasPrefix(p, dir+"/")
}
