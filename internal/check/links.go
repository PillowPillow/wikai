package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func Links(b *wiki.Base) []wiki.Finding {
	l := &linkChecker{b: b, texts: map[string]string{}}
	skip := ""
	if idx := b.Cfg.Index; idx != nil && idx.Check && idx.OtherLinks == "forbid" {
		skip = b.IndexPath()
	}
	for _, p := range b.Pages {
		if p.Path != skip {
			l.scan(p.Path, p.Text, false)
		}
	}
	for _, d := range b.Docs {
		if d.Path != skip {
			l.scan(d.Path, d.Text, d.OutsideOnly)
		}
	}
	return l.findings
}

type linkChecker struct {
	b        *wiki.Base
	texts    map[string]string
	findings []wiki.Finding
}

func (c *linkChecker) add(path string, line int, msg string) {
	c.findings = append(c.findings, wiki.Finding{Check: "links", Path: path, Message: fmt.Sprintf("%s:%d: %s", path, line, msg)})
}

func (c *linkChecker) scan(path, text string, outsideOnly bool) {
	for _, link := range wiki.Links(c.b.Cfg.Links, text) {
		if link.Unparsed {
			if !outsideOnly {
				c.add(path, link.Line, "unparsed link")
			}
			continue
		}
		target, res := c.b.Resolve(path, link)
		switch res {
		case wiki.Outside:
			c.add(path, link.Line, "link leaves the base: "+link.Target)
		case wiki.NotFound:
			if !outsideOnly {
				c.add(path, link.Line, "target not found: "+link.Target)
			}
		case wiki.Resolved:
			if !outsideOnly && link.HasAnchor && strings.HasSuffix(target, ".md") && !c.anchorExists(target, link.Anchor) {
				c.add(path, link.Line, "anchor not found: "+link.Target+"#"+link.Anchor)
			}
		}
	}
}

func (c *linkChecker) targetText(target string) string {
	if p := c.b.Page(target); p != nil {
		return p.Body
	}
	for _, d := range c.b.Docs {
		if d.Path == target {
			return d.Text
		}
	}
	if t, ok := c.texts[target]; ok {
		return t
	}
	data, _ := os.ReadFile(filepath.Join(c.b.Boundary, filepath.FromSlash(target)))
	c.texts[target] = string(data)
	return c.texts[target]
}

func (c *linkChecker) anchorExists(target, anchor string) bool {
	text := c.targetText(target)
	if c.b.Cfg.Links != "wikilink" {
		return wiki.Anchors(text)[anchor]
	}
	if strings.HasPrefix(anchor, "^") {
		return wiki.HasBlockID(text, anchor[1:])
	}
	if i := strings.LastIndex(anchor, "#"); i >= 0 {
		anchor = anchor[i+1:]
	}
	want := foldHeading(anchor)
	for _, h := range wiki.Headings(text) {
		if foldHeading(h) == want {
			return true
		}
	}
	return false
}

func foldHeading(s string) string {
	return strings.ToLower(norm.NFC.String(strings.TrimSpace(s)))
}
