package check

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func scope(b *wiki.Base, folders []string) []*wiki.Page {
	var dirs []string
	for _, f := range folders {
		dirs = append(dirs, norm.NFC.String(b.Cfg.RootPath(f)))
	}
	var out []*wiki.Page
	for _, p := range b.Pages {
		if folders == nil && p.Folder != "" || folders != nil && underAny(p.Path, dirs) {
			out = append(out, p)
		}
	}
	return out
}

func underAny(p string, dirs []string) bool {
	return slices.ContainsFunc(dirs, func(d string) bool {
		return d == "." || p == d || strings.HasPrefix(p, d+"/")
	})
}

func Index(b *wiki.Base) []wiki.Finding {
	idx := b.Cfg.Index
	if idx == nil || !idx.Check {
		return nil
	}
	indexPath := b.IndexPath()
	text, ok := indexText(b, indexPath)
	if !ok {
		return []wiki.Finding{{Check: "index", Path: indexPath, Message: path.Base(indexPath) + " is missing"}}
	}
	var pages []*wiki.Page
	inScope := map[string]bool{}
	for _, p := range scope(b, idx.Include) {
		if p.Path != indexPath {
			pages = append(pages, p)
			inScope[p.Path] = true
		}
	}
	var findings []wiki.Finding
	counts := map[string]int{}
	for _, link := range wiki.Links(b.Cfg.Links, text) {
		target, res := b.Resolve(indexPath, link)
		if res == wiki.Skipped {
			continue
		}
		if res == wiki.Resolved && inScope[target] {
			counts[target]++
			continue
		}
		if idx.OtherLinks == "forbid" {
			findings = append(findings, wiki.Finding{Check: "index", Path: indexPath, Message: "index link resolves to no page: " + link.Target})
		}
	}
	for _, p := range pages {
		switch n := counts[p.Path]; {
		case n == 0:
			findings = append(findings, wiki.Finding{Check: "index", Path: p.Path, Message: "page is not linked from the index"})
		case n > 1:
			findings = append(findings, wiki.Finding{Check: "index", Path: p.Path, Message: fmt.Sprintf("page is linked %d times from the index", n)})
		}
	}
	return findings
}

func indexText(b *wiki.Base, indexPath string) (string, bool) {
	if p := b.Page(indexPath); p != nil {
		return p.Text, true
	}
	for _, d := range b.Docs {
		if d.Path == indexPath {
			return d.Text, true
		}
	}
	return "", false
}
