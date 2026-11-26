package check

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/PillowPillow/wikai/internal/config"
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
	files, err := gitx.LsTree(b.Boundary, ref)
	if err != nil {
		return nil, err
	}
	var findings []wiki.Finding
	for _, raw := range files {
		p := norm.NFC.String(raw)
		var finding string
		switch {
		case slices.ContainsFunc(b.Cfg.AppendOnly, func(a string) bool { return config.SamePath(p, b.Cfg.RootPath(a)) }):
			finding, err = appendOnlyChange(b.Boundary, ref, raw)
		case slices.ContainsFunc(b.Cfg.Frozen, func(f config.Frozen) bool { return frozenAt(b.Cfg, f, p) }):
			finding, err = frozenChange(b.Boundary, ref, raw)
		}
		if err != nil {
			return nil, err
		}
		if finding != "" {
			findings = append(findings, wiki.Finding{Check: "frozen", Path: p, Message: finding})
		}
	}
	return findings, nil
}

func frozenAt(cfg *config.Config, f config.Frozen, p string) bool {
	return config.Under(p, cfg.RootPath(f.Path)) && !slices.ContainsFunc(f.Except, func(e string) bool { return config.Under(p, cfg.RootPath(e)) })
}

func frozenChange(dir, ref, raw string) (string, error) {
	old, err := gitx.Show(dir, ref, raw)
	if err != nil {
		return "", err
	}
	cur, exists, err := readWorking(dir, raw)
	switch {
	case err != nil:
		return "", err
	case !exists:
		return "frozen file deleted", nil
	case !bytes.Equal(old, cur):
		return "frozen file modified", nil
	}
	return "", nil
}

func appendOnlyChange(dir, ref, raw string) (string, error) {
	old, err := gitx.Show(dir, ref, raw)
	if err != nil {
		return "", err
	}
	cur, exists, err := readWorking(dir, raw)
	switch {
	case err != nil:
		return "", err
	case !exists:
		return "append-only file deleted", nil
	case !bytes.HasPrefix(cur, old):
		return "append-only file rewritten", nil
	}
	return "", nil
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
