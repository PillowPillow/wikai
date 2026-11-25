package check

import (
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
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
		for _, name := range pageNames(p) {
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

func pageNames(p *wiki.Page) []string {
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
		if n = strings.ToLower(norm.NFC.String(strings.TrimSpace(n))); n != "" {
			names = append(names, n)
		}
	}
	return names
}
