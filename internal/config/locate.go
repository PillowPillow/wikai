package config

import (
	"os"
	"path/filepath"

	"github.com/PillowPillow/wikai/internal/exit"
)

const fileName = "wikai.yaml"

func Find(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", exit.Usage("%v", err)
	}
	for {
		candidate := filepath.Join(dir, fileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", exit.Usage("no %s found from %s upwards", fileName, startDir)
		}
		dir = parent
	}
}

func Locate(cwd, configFlag, dirFlag string) (*Config, string, error) {
	if configFlag == "" && dirFlag != "" {
		return nil, "", exit.Usage("--dir needs --config")
	}
	file, boundary := configFlag, cwd
	if configFlag == "" {
		found, err := Find(cwd)
		if err != nil {
			return nil, "", err
		}
		file, boundary = found, filepath.Dir(found)
	} else if dirFlag != "" {
		boundary = dirFlag
	}
	file, boundary = absFrom(cwd, file), absFrom(cwd, boundary)
	if info, err := os.Stat(boundary); err != nil || !info.IsDir() {
		return nil, "", exit.Usage("%s is not a folder", boundary)
	}
	cfg, err := Load(file)
	if err != nil {
		return nil, "", err
	}
	return cfg, boundary, nil
}

func absFrom(cwd, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(cwd, p)
}
