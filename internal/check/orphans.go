package check

import (
	"slices"

	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func Orphans(b *wiki.Base) []wiki.Finding {
	o := b.Cfg.Orphans
	if o == nil {
		return nil
	}
	var ignored []string
	for _, p := range o.IgnoreLinksFrom {
		ignored = append(ignored, norm.NFC.String(b.Cfg.RootPath(p)))
	}
	skip := map[string]bool{b.IndexPath(): true, b.LogPath(): true}
	inbound := map[string]bool{}
	for _, p := range b.Pages {
		if skip[p.Path] || underAny(p.Path, ignored) || slices.Contains(o.IgnoreLinksFromTypes, p.Type) {
			continue
		}
		for _, link := range wiki.Links(b.Cfg.Links, p.Text) {
			if target, res := b.Resolve(p.Path, link); res == wiki.Resolved && target != p.Path {
				inbound[target] = true
			}
		}
	}
	var findings []wiki.Finding
	for _, p := range scope(b, o.Scope) {
		if !inbound[p.Path] && !slices.Contains(o.ExemptTypes, p.Type) {
			findings = append(findings, wiki.Finding{Check: "orphans", Path: p.Path, Message: "no other page links to this page"})
		}
	}
	return findings
}
