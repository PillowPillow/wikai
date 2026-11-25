package write

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/gitx"
	"github.com/PillowPillow/wikai/internal/wiki"
)

type MoveOpts struct {
	Old, New string
	NoLog    bool
	Notes    []string
	Date     time.Time
}

type MoveResult struct {
	Modified []string
	Log      string
}

func Move(b *wiki.Base, o MoveOpts) (MoveResult, error) {
	old, dest := nfc(o.Old), nfc(path.Clean(o.New))
	if !b.Git {
		return MoveResult{}, exit.Usage("%s is not a git repository", b.Boundary)
	}
	page := b.Page(old)
	if page == nil || old == b.IndexPath() || old == b.LogPath() {
		return MoveResult{}, exit.Usage("%s is not a page", o.Old)
	}
	if page.Frozen {
		return MoveResult{}, exit.Refused("%s is frozen: it is never renamed", old)
	}
	root := nfc(b.Cfg.RootPath("."))
	if !strings.HasSuffix(dest, ".md") || path.IsAbs(dest) || dest == ".." || strings.HasPrefix(dest, "../") || !inDir(dest, root) {
		return MoveResult{}, exit.Usage("%s must be a .md path inside %s", o.New, root)
	}
	stem := strings.TrimSuffix(path.Base(dest), ".md")
	style := b.Cfg.Links
	if style == "wikilink" && strings.ContainsAny(stem, "#|[]") {
		return MoveResult{}, exit.Usage("%s: a wikilink name holds none of # | [ ]", o.New)
	}
	if reserved(b, dest) {
		return MoveResult{}, exit.Refused("%s is the index, the log, a skeleton or an ignored path", dest)
	}
	if _, err := os.Lstat(diskPath(b, dest)); err == nil {
		return MoveResult{}, exit.Refused("%s exists", dest)
	}
	archive := frozenDest(b, dest)
	if !archive && !typeAllows(b, page.Type, dest) {
		return MoveResult{}, exit.Refused("a page moves only inside its type or to a frozen path")
	}

	day := o.Date
	if day.IsZero() {
		day = time.Now()
	}
	logOpts := LogOpts{
		Operation: "move",
		Title:     fmt.Sprintf(Locale(b.Cfg.Locale).MoveTitle, old, dest),
		Slug:      Slugify(stem),
		Created:   []string{dest},
		Notes:     append([]string{fmt.Sprintf(Locale(b.Cfg.Locale).MoveNote, old)}, o.Notes...),
		Date:      day,
	}
	writeLog := !o.NoLog && b.Cfg.Log != nil
	if writeLog {
		if err := checkMoveLog(b, logOpts); err != nil {
			return MoveResult{}, err
		}
	}

	rewrite := linkRewriter(b, old, dest)
	changes := map[string]string{}
	var modified []string
	for _, p := range b.Pages {
		if p.Path == b.IndexPath() || p.Frozen {
			continue
		}
		text := wiki.RewriteLinks(style, p.Text, rewrite(p.Path))
		switch {
		case p.Path == old:
			changes[dest] = text
		case text != p.Text:
			changes[p.Path] = text
			modified = append(modified, p.Path)
		}
	}
	if err := updateIndex(b, old, dest, archive, rewrite, changes); err != nil {
		return MoveResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(diskPath(b, dest)), 0o755); err != nil {
		return MoveResult{}, err
	}
	if err := gitx.Mv(b.Boundary, old, dest); err != nil {
		return MoveResult{}, err
	}
	for _, p := range slices.Sorted(mapKeys(changes)) {
		if err := os.WriteFile(diskPath(b, p), []byte(changes[p]), 0o644); err != nil {
			return MoveResult{}, err
		}
	}
	res := MoveResult{Modified: modified}
	if writeLog {
		logOpts.Modified = modified
		var err error
		if res.Log, err = NewLog(b, logOpts); err != nil {
			return res, err
		}
	}
	return res, nil
}

func diskPath(b *wiki.Base, p string) string {
	return filepath.Join(b.Boundary, filepath.FromSlash(p))
}

func reserved(b *wiki.Base, dest string) bool {
	cfg := b.Cfg
	if dest == b.IndexPath() || dest == b.LogPath() {
		return true
	}
	if cfg.Log != nil && cfg.Log.Mode == "dir" && inDir(dest, b.LogPath()) {
		return true
	}
	if cfg.Skeletons != "" && inDir(dest, nfc(cfg.RootPath(cfg.Skeletons))) {
		return true
	}
	return slices.ContainsFunc(cfg.Ignore, func(i string) bool { return inDir(dest, nfc(cfg.RootPath(i))) })
}

func frozenDest(b *wiki.Base, dest string) bool {
	for _, f := range b.Cfg.Frozen {
		if !inDir(dest, nfc(b.Cfg.RootPath(f.Path))) {
			continue
		}
		if !slices.ContainsFunc(f.Except, func(e string) bool { return inDir(dest, nfc(b.Cfg.RootPath(e))) }) {
			return true
		}
	}
	return false
}

// A type without folders may sit anywhere, except in the folder of another type.
func typeAllows(b *wiki.Base, typeName, dest string) bool {
	inFolders := func(folders []string) bool {
		return slices.ContainsFunc(folders, func(f string) bool { return inDir(dest, nfc(b.Cfg.RootPath(f))) })
	}
	if t, ok := b.Cfg.Types[typeName]; ok && len(t.Folders) > 0 {
		return inFolders(t.Folders)
	}
	for _, t := range b.Cfg.Types {
		if inFolders(t.Folders) {
			return false
		}
	}
	return true
}

func checkMoveLog(b *wiki.Base, o LogOpts) error {
	l := b.Cfg.Log
	if !slices.Contains(l.Operations, o.Operation) {
		return exit.Usage("operation %q is not one of: %s", o.Operation, strings.Join(l.Operations, ", "))
	}
	for _, s := range append([]string{o.Title}, o.Notes...) {
		if _, err := oneLine("the log entry", s); err != nil {
			return err
		}
	}
	if l.Mode == "dir" {
		if o.Slug == "" {
			return exit.Usage("%s gives an empty log slug: pass --no-log or choose another name", o.Created[0])
		}
		file := nfc(path.Join(b.LogPath(), o.Date.Format("2006-01-02")+"-"+o.Operation+"-"+o.Slug+".md"))
		if _, err := os.Lstat(diskPath(b, file)); err == nil {
			return exit.Refused("%s exists: log entries are never modified, pass --no-log or choose another name", file)
		}
		return nil
	}
	if _, err := os.Stat(diskPath(b, b.LogPath())); err != nil {
		return exit.Usage("%s is missing", b.LogPath())
	}
	return nil
}

// Resolution uses the pre-move base only: its wikilink cache does not see the moved file.
func linkRewriter(b *wiki.Base, old, dest string) func(from string) func(wiki.Link) (string, bool) {
	newDir := path.Dir(dest)
	rootRel := strings.TrimPrefix(dest, b.Cfg.Root+"/")
	if b.Cfg.Root == "." {
		rootRel = dest
	}
	pathRef := strings.TrimSuffix(rootRel, ".md")
	oldStem := strings.TrimSuffix(path.Base(old), ".md")
	newStem := strings.TrimSuffix(path.Base(dest), ".md")
	stemRef := newStem
	if stemTaken(b, old, newStem) {
		stemRef = pathRef
	}
	return func(from string) func(wiki.Link) (string, bool) {
		return func(l wiki.Link) (string, bool) {
			if l.Target == "" {
				return "", false
			}
			target, r := b.Resolve(from, l)
			if b.Cfg.Links == "wikilink" {
				switch {
				case r != wiki.Resolved || target != old:
					return "", false
				case strings.Contains(l.Target, "/"):
					return pathRef, true
				case oldStem != newStem:
					return stemRef, true
				}
				return "", false
			}
			fromDir := path.Dir(from)
			if from == old {
				fromDir = newDir
			}
			switch {
			case r == wiki.Resolved && target == old:
				return relPath(fromDir, dest), true
			case from == old && newDir != path.Dir(old) && (r == wiki.Resolved || r == wiki.NotFound):
				return relPath(fromDir, target), true
			}
			return "", false
		}
	}
}

func stemTaken(b *wiki.Base, old, stem string) bool {
	key := strings.ToLower(nfc(stem))
	for f := range b.Files {
		if f != old && strings.TrimSuffix(strings.ToLower(path.Base(f)), ".md") == key {
			return true
		}
	}
	return false
}

func updateIndex(b *wiki.Base, old, dest string, archive bool, rewrite func(string) func(wiki.Link) (string, bool), changes map[string]string) error {
	idxPath := b.IndexPath()
	if idxPath == "" {
		return nil
	}
	oldHeading, oldFlat, oldOK := indexGroup(b, old)
	newHeading, newFlat, newOK := indexGroup(b, dest)
	raw, err := os.ReadFile(diskPath(b, idxPath))
	if errors.Is(err, fs.ErrNotExist) {
		if oldOK || newOK {
			return exit.Usage("%s is missing", idxPath)
		}
		return nil
	} else if err != nil {
		return err
	}
	style := b.Cfg.Links
	text := string(raw)
	regroup := oldOK && (archive || !newOK || oldHeading != newHeading || oldFlat != newFlat)
	var line string
	if regroup {
		line = indexLine(b, text, old)
		if text, err = IndexRemove(b, text, old); err != nil {
			return err
		}
	}
	text = wiki.RewriteLinks(style, text, rewrite(idxPath))
	switch {
	case archive || !newOK:
	case regroup && line != "":
		line = wiki.RewriteLinks(style, line, rewrite(idxPath))
		target := relPath(path.Dir(idxPath), dest)
		if style == "wikilink" {
			target = strings.TrimSuffix(path.Base(dest), ".md")
		}
		text, err = rewriteIndex(b, text, dest, target, func(entries []string) []string { return append(entries, line) })
	default:
		text, err = IndexResort(b, text, dest)
	}
	if err != nil {
		return err
	}
	if text != string(raw) {
		changes[idxPath] = text
	}
	return nil
}

func indexLine(b *wiki.Base, index, pagePath string) string {
	for _, l := range strings.Split(index, "\n") {
		if t, ok := entryTarget(b.Cfg.Links, l); ok && sameTarget(b, b.IndexPath(), t, pagePath) {
			return l
		}
	}
	return ""
}
