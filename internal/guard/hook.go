package guard

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/PillowPillow/wikai/internal/config"
	"golang.org/x/text/unicode/norm"
)

type Event struct {
	ToolName  string `json:"tool_name"`
	Cwd       string `json:"cwd"`
	ToolInput struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
		Content      string `json:"content"`
	} `json:"tool_input"`
}

func Hook(cfg *config.Config, boundary string, ev Event) (deny string) {
	target := ev.ToolInput.FilePath
	if target == "" {
		target = ev.ToolInput.NotebookPath
	}
	if target == "" {
		return ""
	}
	if !filepath.IsAbs(target) {
		if !filepath.IsAbs(ev.Cwd) {
			return fmt.Sprintf("%s: relative path without an absolute cwd", target)
		}
		target = ev.Cwd + string(filepath.Separator) + target
	}
	root, err := resolve(boundary, "")
	if err != nil {
		return fmt.Sprintf("%s: cannot resolve the base: %v", boundary, err)
	}
	abs, err := resolve(target, root)
	if err != nil {
		return fmt.Sprintf("%s: cannot resolve the path: %v", target, err)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	rel = norm.NFC.String(filepath.ToSlash(rel))
	switch Classify(cfg, rel) {
	case Protected:
		return rel + ": protected path, propose the change to the human"
	case Frozen:
		if _, err := os.Lstat(abs); !errors.Is(err, fs.ErrNotExist) {
			return rel + ": frozen file, existing files are never modified"
		}
	case AppendOnly:
		cur, err := os.ReadFile(abs)
		if errors.Is(err, fs.ErrNotExist) {
			return ""
		}
		if err != nil || ev.ToolName != "Write" || !bytes.HasPrefix([]byte(ev.ToolInput.Content), cur) {
			return rel + ": append-only file, new content must start with the current content"
		}
	}
	return ""
}

// Resolves p as the kernel would: symlinks are followed before "..", missing components are kept
// as given, and each existing component takes its on-disk spelling (case-insensitive filesystems).
// A folder that cannot be listed fails the resolution only inside strict, where the spelling decides a rule.
func resolve(p, strict string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%s is not absolute", p)
	}
	queue := strings.Split(p, string(filepath.Separator))
	resolved := string(filepath.Separator)
	for links := 0; len(queue) > 0; {
		c := queue[0]
		queue = queue[1:]
		switch c {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}
		next := filepath.Join(resolved, c)
		info, err := os.Lstat(next)
		if errors.Is(err, fs.ErrNotExist) {
			resolved = next
			continue
		}
		if err != nil {
			return "", err
		}
		if name, err := onDiskName(resolved, c, info); err == nil {
			c = name
		} else if strict != "" && within(resolved, strict) {
			return "", err
		}
		next = filepath.Join(resolved, c)
		if info.Mode()&fs.ModeSymlink == 0 {
			resolved = next
			continue
		}
		if links++; links > 255 {
			return "", fmt.Errorf("%s: too many symlinks", p)
		}
		dest, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(dest) {
			resolved = string(filepath.Separator)
		}
		queue = append(strings.Split(dest, string(filepath.Separator)), queue...)
	}
	return resolved, nil
}

func within(p, dir string) bool {
	return p == dir || strings.HasPrefix(p, strings.TrimSuffix(dir, string(filepath.Separator))+string(filepath.Separator))
}

func onDiskName(dir, name string, info fs.FileInfo) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name() == name {
			return name, nil
		}
	}
	want := norm.NFC.String(name)
	for _, e := range entries {
		if !strings.EqualFold(norm.NFC.String(e.Name()), want) {
			continue
		}
		if other, err := os.Lstat(filepath.Join(dir, e.Name())); err == nil && os.SameFile(info, other) {
			return e.Name(), nil
		}
	}
	return name, nil
}
