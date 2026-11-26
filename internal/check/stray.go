package check

import "github.com/PillowPillow/wikai/internal/wiki"

func Stray(b *wiki.Base) []wiki.Finding {
	var findings []wiki.Finding
	for _, p := range b.Stray {
		findings = append(findings, wiki.Finding{Check: "stray", Path: p, Message: "markdown file outside the page folders"})
	}
	return findings
}
