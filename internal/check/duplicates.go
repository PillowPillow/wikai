package check

import (
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
)

func Duplicates(b *wiki.Base) []wiki.Finding {
	var findings []wiki.Finding
	owners := map[string]map[string]bool{}
	for _, p := range b.Pages {
		if p.Title == "" {
			if b.Cfg.Title == "heading" {
				findings = append(findings, wiki.Finding{Check: "duplicates", Path: p.Path, Message: "no title"})
			}
			continue
		}
		for _, name := range wiki.Names(p) {
			if owners[name] == nil {
				owners[name] = map[string]bool{}
			}
			owners[name][p.Path] = true
		}
	}
	shared := map[string][]string{}
	for name, paths := range owners {
		if len(paths) > 1 {
			for p := range paths {
				shared[p] = append(shared[p], name)
			}
		}
	}
	for _, p := range b.Pages {
		if names := shared[p.Path]; names != nil {
			slices.Sort(names)
			findings = append(findings, wiki.Finding{Check: "duplicates", Path: p.Path, Message: "name shared with another page: " + strings.Join(names, ", ")})
		}
	}
	return findings
}
