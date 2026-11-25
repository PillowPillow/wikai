package check

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/gitx"
	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

func Frozen(b *wiki.Base, ref string) ([]wiki.Finding, error) {
	if ref == "" {
		return nil, nil
	}
	if err := gitx.VerifyRef(b.Boundary, ref); err != nil {
		return nil, exit.Usage("cannot read git ref %q", ref)
	}
	var findings []wiki.Finding
	seen := map[string]bool{}
	for _, f := range b.Cfg.Frozen {
		var except []string
		for _, e := range f.Except {
			except = append(except, norm.NFC.String(b.Cfg.RootPath(e)))
		}
		files, err := gitx.LsTree(b.Boundary, ref, b.Cfg.RootPath(f.Path))
		if err != nil {
			return nil, err
		}
		for _, raw := range files {
			p := norm.NFC.String(raw)
			if seen[p] || slices.ContainsFunc(except, func(e string) bool { return p == e || strings.HasPrefix(p, e+"/") }) {
				continue
			}
			seen[p] = true
			old, err := gitx.Show(b.Boundary, ref, raw)
			if err != nil {
				return nil, err
			}
			cur, exists, err := readWorking(b.Boundary, raw)
			if err != nil {
				return nil, err
			}
			switch {
			case !exists:
				findings = append(findings, wiki.Finding{Check: "frozen", Path: p, Message: "frozen file deleted"})
			case !bytes.Equal(old, cur):
				findings = append(findings, wiki.Finding{Check: "frozen", Path: p, Message: "frozen file modified"})
			}
		}
	}
	for _, a := range b.Cfg.AppendOnly {
		raw := b.Cfg.RootPath(a)
		files, err := gitx.LsTree(b.Boundary, ref, raw)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(files, raw) {
			continue
		}
		old, err := gitx.Show(b.Boundary, ref, raw)
		if err != nil {
			return nil, err
		}
		cur, exists, err := readWorking(b.Boundary, raw)
		if err != nil {
			return nil, err
		}
		p := norm.NFC.String(raw)
		switch {
		case !exists:
			findings = append(findings, wiki.Finding{Check: "frozen", Path: p, Message: "append-only file deleted"})
		case !bytes.HasPrefix(cur, old):
			findings = append(findings, wiki.Finding{Check: "frozen", Path: p, Message: "append-only file rewritten"})
		}
	}
	return findings, nil
}

func readWorking(dir, rel string) ([]byte, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		data, err = os.ReadFile(filepath.Join(dir, filepath.FromSlash(norm.NFC.String(rel))))
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}
