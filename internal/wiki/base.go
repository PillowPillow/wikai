package wiki

import (
	"cmp"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/gitx"
	"golang.org/x/text/unicode/norm"
)

type Finding struct {
	Check   string `json:"check"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Page struct {
	Path, Text, Body, RawMeta string
	Meta                      map[string]any
	MetaErr                   error
	Title                     string
	Type                      string
	Folder                    string
	Frozen, Unchecked         bool
}

type Doc struct {
	Path, Text  string
	OutsideOnly bool
}

type Base struct {
	Boundary     string
	Cfg          *config.Config
	Git          bool
	Files        map[string]bool
	Pages        []*Page
	Docs         []*Doc
	Stray        []string
	LoadFindings []Finding

	wikilinkTargets map[string]string
}

type Resolution int

const (
	Resolved Resolution = iota
	Skipped
	Outside
	NotFound
)

func nfc(s string) string { return norm.NFC.String(s) }

type frozenPath struct {
	dir     string
	except  []string
	checked bool
}

type typeFolder struct{ rel, full string }

type layout struct {
	ignore, unchecked []string
	skeletons, logDir string
	special           []string
	frozen            []frozenPath
	folders           []typeFolder
}

func newLayout(cfg *config.Config) *layout {
	rp := func(p string) string { return nfc(cfg.RootPath(p)) }
	rps := func(list []string) []string {
		var out []string
		for _, p := range list {
			out = append(out, rp(p))
		}
		return out
	}
	l := &layout{ignore: rps(cfg.Ignore), unchecked: rps(cfg.UncheckedFrontmatter), skeletons: rp(cfg.Skeletons)}
	if cfg.Index != nil {
		l.special = append(l.special, rp(cfg.Index.File))
	}
	if cfg.Log != nil {
		if cfg.Log.Mode == "dir" {
			l.logDir = rp(cfg.Log.Path)
		} else {
			l.special = append(l.special, rp(cfg.Log.Path))
		}
	}
	for _, f := range cfg.Frozen {
		l.frozen = append(l.frozen, frozenPath{rp(f.Path), rps(f.Except), f.Checked})
	}
	for _, t := range cfg.Types {
		for _, f := range t.Folders {
			l.folders = append(l.folders, typeFolder{nfc(path.Clean(f)), rp(f)})
		}
	}
	return l
}

func (l *layout) frozenAt(p string) *frozenPath {
	for i, f := range l.frozen {
		if config.Under(p, f.dir) && !config.UnderAny(p, f.except) {
			return &l.frozen[i]
		}
	}
	return nil
}

func (l *layout) folderOf(p string) string {
	best, bestLen := "", -1
	for _, f := range l.folders {
		if config.Under(p, f.full) && len(f.full) > bestLen {
			best, bestLen = f.rel, len(f.full)
		}
	}
	return best
}

func Load(boundary string, cfg *config.Config) (*Base, error) {
	b := &Base{Boundary: boundary, Cfg: cfg, Git: gitx.IsRepo(boundary)}
	l := newLayout(cfg)
	root := filepath.Join(boundary, filepath.FromSlash(cfg.Root))
	err := filepath.WalkDir(root, func(disk string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(boundary, disk)
		if err != nil {
			return err
		}
		p := nfc(filepath.ToSlash(rel))
		if d.Name() == ".git" || config.UnderAny(p, l.ignore) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		return b.add(l, disk, p)
	})
	if err != nil {
		return nil, err
	}
	if err := b.loadFiles(); err != nil {
		return nil, err
	}
	slices.SortFunc(b.Pages, func(x, y *Page) int { return cmp.Compare(x.Path, y.Path) })
	slices.SortFunc(b.Docs, func(x, y *Doc) int { return cmp.Compare(x.Path, y.Path) })
	slices.Sort(b.Stray)
	slices.SortFunc(b.LoadFindings, func(x, y Finding) int { return cmp.Compare(x.Path, y.Path) })
	return b, nil
}

func (b *Base) add(l *layout, disk, p string) error {
	if config.Under(p, l.skeletons) || l.logDir != "" && config.Under(p, l.logDir) {
		return nil
	}
	data, err := os.ReadFile(disk)
	if err != nil {
		return err
	}
	if !utf8.Valid(data) {
		b.LoadFindings = append(b.LoadFindings, Finding{"frontmatter", p, "not UTF-8"})
		return nil
	}
	text := string(data)
	frozen := l.frozenAt(p)
	if frozen != nil && !frozen.checked {
		b.Docs = append(b.Docs, &Doc{Path: p, Text: text, OutsideOnly: true})
		return nil
	}
	folder := l.folderOf(p)
	if slices.Contains(l.special, p) {
		if _, ok := findFrontmatter(text); !ok {
			b.Docs = append(b.Docs, &Doc{Path: p, Text: text})
			return nil
		}
	} else if folder == "" && frozen == nil && b.Cfg.Stray {
		b.Stray = append(b.Stray, p)
		return nil
	}
	meta, raw, body, metaErr := ParseFrontmatter(text)
	page := &Page{Path: p, Text: text, Body: body, RawMeta: raw, Meta: meta, MetaErr: metaErr,
		Folder: folder, Frozen: frozen != nil, Unchecked: config.UnderAny(p, l.unchecked)}
	page.Type, _ = meta["type"].(string)
	if b.Cfg.Title == "filename" {
		page.Title = strings.TrimSuffix(path.Base(p), ".md")
	} else {
		page.Title, _ = HeadingTitle(body)
	}
	b.Pages = append(b.Pages, page)
	return nil
}

func (b *Base) loadFiles() error {
	b.Files = map[string]bool{}
	if b.Cfg.Links == "markdown" && b.Git {
		tracked, err := gitx.LsFiles(b.Boundary)
		if err != nil {
			return err
		}
		for _, f := range tracked {
			if _, err := os.Lstat(filepath.Join(b.Boundary, filepath.FromSlash(f))); err == nil {
				b.Files[nfc(f)] = true
			}
		}
		return nil
	}
	return filepath.WalkDir(b.Boundary, func(disk string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(b.Boundary, disk)
		if err != nil {
			return err
		}
		b.Files[nfc(filepath.ToSlash(rel))] = true
		return nil
	})
}

func (b *Base) Page(p string) *Page {
	p = nfc(p)
	i, ok := slices.BinarySearchFunc(b.Pages, p, func(x *Page, p string) int { return cmp.Compare(x.Path, p) })
	if !ok {
		return nil
	}
	return b.Pages[i]
}

func (b *Base) IndexPath() string {
	if b.Cfg.Index == nil {
		return ""
	}
	return nfc(b.Cfg.RootPath(b.Cfg.Index.File))
}

func (b *Base) LogPath() string {
	if b.Cfg.Log == nil {
		return ""
	}
	return nfc(b.Cfg.RootPath(b.Cfg.Log.Path))
}

func (b *Base) Resolve(from string, l Link) (target string, r Resolution) {
	if l.External || l.Unparsed {
		return "", Skipped
	}
	if l.Target == "" {
		return from, Resolved
	}
	if b.Cfg.Links == "wikilink" {
		return b.resolveWikilink(l.Target)
	}
	t := nfc(l.Target)
	if path.IsAbs(t) {
		return "", Outside
	}
	p := path.Join(path.Dir(from), t)
	if p == ".." || strings.HasPrefix(p, "../") {
		return "", Outside
	}
	if !b.Files[p] {
		return p, NotFound
	}
	return p, Resolved
}

func (b *Base) resolveWikilink(target string) (string, Resolution) {
	if b.wikilinkTargets == nil {
		b.wikilinkTargets = wikilinkTargets(b.Files)
	}
	key := strings.TrimSuffix(strings.ToLower(nfc(target)), ".md")
	if p, ok := b.wikilinkTargets[key]; ok {
		return p, Resolved
	}
	return "", NotFound
}

// Every "/"-suffix of the path, with and without .md, is a key, so stems and path suffixes need no scan.
func wikilinkTargets(files map[string]bool) map[string]string {
	targets := map[string]string{}
	keep := func(key, file string) {
		cur, ok := targets[key]
		if !ok || len(file) < len(cur) || len(file) == len(cur) && file < cur {
			targets[key] = file
		}
	}
	for file := range files {
		lower := strings.ToLower(file)
		for _, form := range []string{lower, strings.TrimSuffix(lower, ".md")} {
			keep(form, file)
			for i := range len(form) {
				if form[i] == '/' {
					keep(form[i+1:], file)
				}
			}
		}
	}
	return targets
}

func (b *Base) Rel(cwd, arg string) (string, error) {
	abs := filepath.Clean(arg)
	if !filepath.IsAbs(arg) {
		abs = filepath.Join(cwd, arg)
	}
	abs = filepath.Join(realPath(filepath.Dir(abs)), filepath.Base(abs))
	rel, err := filepath.Rel(realPath(b.Boundary), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", exit.Usage("%s is outside the base %s", arg, b.Boundary)
	}
	return nfc(filepath.ToSlash(rel)), nil
}

// Resolves the longest existing prefix, so /var and /private/var compare equal for a file not yet created.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	dir := filepath.Dir(p)
	if dir == p {
		return p
	}
	return filepath.Join(realPath(dir), filepath.Base(p))
}
