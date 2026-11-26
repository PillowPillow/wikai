package preset

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/marker"
)

//go:embed all:home-repo all:obsidian-vault
var presets embed.FS

func Names() []string { return []string{"home-repo", "obsidian-vault"} }

type file struct {
	rel  string
	data []byte
}

func Write(name, dir string, date time.Time) ([]string, error) {
	if !slices.Contains(Names(), name) {
		return nil, exit.Usage("unknown preset %q: choose one of %s", name, strings.Join(Names(), ", "))
	}
	var files []file
	err := fs.WalkDir(presets, name, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := presets.ReadFile(p)
		files = append(files, file{rel: strings.TrimPrefix(p, name+"/"), data: data})
		return err
	})
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(filepath.Join(dir, "wikai.yaml")); err == nil {
		return nil, exit.Refused("wikai.yaml exists")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	var created []string
	put := func(rel string, data []byte) error {
		disk := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(disk), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(disk, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return nil
		} else if err != nil {
			return err
		}
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			created = append(created, rel)
		}
		return err
	}

	for _, f := range files {
		if f.rel == "wikai.yaml" {
			if err := put(f.rel, f.data); err != nil {
				return nil, err
			}
		}
	}
	cfg, err := config.Load(filepath.Join(dir, "wikai.yaml"))
	if err != nil {
		return created, err
	}
	skeletons := cfg.RootPath(cfg.Skeletons)
	for _, f := range files {
		if f.rel == "wikai.yaml" {
			continue
		}
		data := f.data
		if !strings.HasPrefix(f.rel, skeletons+"/") {
			text, err := marker.Fill(string(data), nil, date, cfg.Locale)
			if err != nil {
				return created, err
			}
			data = []byte(text)
		}
		if err := put(f.rel, data); err != nil {
			return created, err
		}
	}

	var keep []string
	for _, t := range cfg.Types {
		keep = append(keep, t.Folders...)
	}
	if cfg.Log != nil && cfg.Log.Mode == "dir" {
		keep = append(keep, cfg.Log.Path)
	}
	for _, f := range cfg.Frozen {
		keep = append(keep, f.Path)
	}
	slices.Sort(keep)
	for _, folder := range slices.Compact(keep) {
		rel := path.Join(cfg.RootPath(folder), ".gitkeep")
		if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil {
			continue
		}
		if err := put(rel, nil); err != nil {
			return created, err
		}
	}
	return created, nil
}
