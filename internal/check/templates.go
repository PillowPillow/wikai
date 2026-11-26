package check

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/PillowPillow/wikai/internal/marker"
	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func Templates(b *wiki.Base) []wiki.Finding {
	findings := skeletonFormats(b)
	if len(b.Cfg.TemplateMarkers) == 0 {
		return findings
	}
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

func skeletonFormats(b *wiki.Base) []wiki.Finding {
	folder := b.Cfg.RootPath(b.Cfg.Skeletons)
	var findings []wiki.Finding
	_ = filepath.WalkDir(filepath.Join(b.Boundary, filepath.FromSlash(folder)), func(p string, d fs.DirEntry, err error) error {
		rel, _ := filepath.Rel(b.Boundary, p)
		at := norm.NFC.String(filepath.ToSlash(rel))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		var data []byte
		if err == nil {
			if d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, err = os.ReadFile(p)
		}
		if err != nil {
			var pe *fs.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			findings = append(findings, wiki.Finding{Check: "templates", Path: at, Message: "not readable: " + err.Error()})
			return nil
		}
		for _, e := range marker.Check(string(data)) {
			findings = append(findings, wiki.Finding{Check: "templates", Path: at, Message: e.Error()})
		}
		return nil
	})
	return findings
}
