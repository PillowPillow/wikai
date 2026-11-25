package check

import (
	"strings"

	"github.com/PillowPillow/wikai/internal/wiki"
)

func Templates(b *wiki.Base) []wiki.Finding {
	if len(b.Cfg.TemplateMarkers) == 0 {
		return nil
	}
	var findings []wiki.Finding
	for _, p := range b.Pages {
		text := p.RawMeta
		if !p.Frozen {
			var body strings.Builder
			for _, l := range wiki.OutsideCode(p.Body) {
				body.WriteString(l.Text)
				body.WriteByte('\n')
			}
			text += "\n" + body.String()
		}
		for _, m := range b.Cfg.TemplateMarkers {
			if strings.Contains(text, m) {
				findings = append(findings, wiki.Finding{Check: "templates", Path: p.Path, Message: "template marker left: " + m})
			}
		}
	}
	return findings
}
