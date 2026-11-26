package write

import (
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
	"github.com/PillowPillow/wikai/internal/wiki"
)

type PageOpts struct {
	Kind, Title, Summary, Slug string
	Date                       time.Time
	Set                        map[string]string
}

func oneLine(label, s string) (string, error) {
	if strings.ContainsAny(s, "\r\n") {
		return "", exit.Usage("%s must hold no line break", label)
	}
	return s, nil
}

const wikilinkNameMsg = "%s: a wikilink name holds none of # | [ ]"

func breaksWikilink(style, name string) bool {
	return style == "wikilink" && strings.ContainsAny(name, "#|[]")
}

func NewPage(b *wiki.Base, o PageOpts) (string, error) {
	_, kind, ok := b.Cfg.KindOf(o.Kind)
	if !ok {
		return "", exit.Usage("unknown kind %s", o.Kind)
	}
	if kind.Folder == "" {
		return "", exit.Usage("kind %s has no folder", o.Kind)
	}
	title, err := oneLine("the title", strings.TrimSpace(o.Title))
	if err != nil {
		return "", err
	}
	if title == "" {
		return "", exit.Usage("the title is empty")
	}
	summary := strings.Join(strings.Fields(o.Summary), " ")
	day := o.Date
	if day.IsZero() {
		day = time.Now()
	}
	date := day.Format("2006-01-02")
	slug := o.Slug
	if slug == "" {
		slug = Slugify(title)
	}
	if slug == "" && strings.Contains(kind.Name, "{slug}") {
		return "", exit.Usage("the title gives an empty slug: pass --slug")
	}
	name := nfc(strings.NewReplacer("{title}", title, "{slug}", slug, "{date:compact}", day.Format("20060102"), "{date}", date).Replace(kind.Name))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") {
		return "", exit.Usage("the page name %q is not a file name", name)
	}
	if breaksWikilink(b.Cfg.Links, name) {
		return "", exit.Usage(wikilinkNameMsg, name)
	}
	pagePath := nfc(path.Join(b.Cfg.RootPath(kind.Folder), name+".md"))
	disk := filepath.Join(b.Boundary, filepath.FromSlash(pagePath))
	if _, err := os.Lstat(disk); err == nil || b.Page(pagePath) != nil {
		return "", exit.Refused("%s exists", pagePath)
	}
	pageTitle := title
	if b.Cfg.Title == "filename" {
		pageTitle = name
	}
	for _, p := range b.Pages {
		if slices.Contains(wiki.Names(p), wiki.FoldName(pageTitle)) {
			return "", exit.Refused("%q is already the title or an alias of %s: update that page", pageTitle, p.Path)
		}
	}
	if owner := stemTaken(b, "", name); b.Cfg.Links == "wikilink" && owner != "" {
		return "", exit.Refused("%s already has the name %s: a [[%s]] link would not reach one of them", owner, name, name)
	}

	skeleton := kind.Skeleton
	if skeleton == "" {
		skeleton = o.Kind
	}
	if !fs.ValidPath(skeleton) {
		return "", exit.Usage("skeleton name %q is not valid", skeleton)
	}
	skelPath := filepath.Join(b.Boundary, filepath.FromSlash(b.Cfg.RootPath(b.Cfg.Skeletons)), filepath.FromSlash(skeleton)+".md")
	data, err := os.ReadFile(skelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", exit.Usage("skeleton %s is missing", path.Join(b.Cfg.RootPath(b.Cfg.Skeletons), skeleton+".md"))
	} else if err != nil {
		return "", err
	}
	values := map[string]string{"title": title, "slug": slug, "summary": summary}
	for _, k := range slices.Sorted(mapKeys(o.Set)) {
		v, err := oneLine("--set "+k, o.Set[k])
		if err != nil {
			return "", err
		}
		if _, builtin := values[k]; !builtin {
			values[k] = v
		}
	}
	text, err := marker.Fill(string(data), values, day)
	if err != nil {
		return "", exit.Usage("skeleton %s: %v", skeleton, err)
	}
	if text, err = setFrontmatter(text, kind.Set); err != nil {
		return "", exit.Usage("skeleton %s: %v", skeleton, err)
	}
	text = fillDates(text, requiredDates(b.Cfg, o.Kind), date)
	text = fillEmptySections(text, Locale(b.Cfg.Locale).EmptySection)

	if err := RequireGroup(b, pagePath); err != nil {
		return "", err
	}
	var index, indexDisk string
	if Indexed(b, pagePath) {
		indexDisk = filepath.Join(b.Boundary, filepath.FromSlash(b.IndexPath()))
		raw, err := os.ReadFile(indexDisk)
		if errors.Is(err, fs.ErrNotExist) {
			return "", exit.Usage("%s is missing", b.IndexPath())
		} else if err != nil {
			return "", err
		}
		if index, err = IndexAdd(b, string(raw), pagePath, title, summary); err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(filepath.Dir(disk), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(disk, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return "", exit.Refused("%s exists", pagePath)
	} else if err != nil {
		return "", err
	}
	_, err = f.WriteString(text)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	if indexDisk != "" {
		if err := writeKeepingMode(indexDisk, index); err != nil {
			return "", err
		}
	}
	return pagePath, nil
}

func writeKeepingMode(file, text string) error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(file); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(file, []byte(text), mode)
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func frontmatterEnd(lines []string) int {
	if len(lines) == 0 || strings.TrimPrefix(lines[0], "\ufeff") != "---" {
		return -1
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return i
		}
	}
	return -1
}

func setFrontmatter(text string, set map[string]string) (string, error) {
	if len(set) == 0 {
		return text, nil
	}
	lines := strings.Split(text, "\n")
	end := frontmatterEnd(lines)
	if end < 0 {
		return "", errors.New("has no frontmatter to write the set values into")
	}
	for _, k := range slices.Sorted(mapKeys(set)) {
		v, err := oneLine("set "+k, set[k])
		if err != nil {
			return "", err
		}
		line := k + ": " + v
		if i := slices.IndexFunc(lines[1:end], func(l string) bool { return strings.HasPrefix(l, k+":") }); i >= 0 {
			lines[i+1] = line
			continue
		}
		lines = slices.Insert(lines, end, line)
		end++
	}
	return strings.Join(lines, "\n"), nil
}

func requiredDates(cfg *config.Config, kindName string) map[string]bool {
	typeName, kind, _ := cfg.KindOf(kindName)
	rules := map[string]config.Rule{}
	for _, layer := range []map[string]config.Rule{cfg.Frontmatter, cfg.Types[typeName].Frontmatter, kind.Frontmatter} {
		for k, r := range layer {
			rules[k] = r
		}
	}
	out := map[string]bool{}
	for k, r := range rules {
		if r.Required && r.Format == "date" && !r.EmptyAllowed {
			out[k] = true
		}
	}
	return out
}

func fillDates(text string, keys map[string]bool, date string) string {
	if len(keys) == 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	end := frontmatterEnd(lines)
	for i := 1; i < end; i++ {
		k, rest, found := strings.Cut(lines[i], ":")
		if found && keys[k] && strings.Trim(rest, " \t") == "" {
			lines[i] = k + ": " + date
		}
	}
	return strings.Join(lines, "\n")
}

func fillEmptySections(text, empty string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for i, l := range lines {
		out = append(out, l)
		if !strings.HasPrefix(l, "## ") {
			continue
		}
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j == len(lines) || strings.HasPrefix(lines[j], "## ") {
			out = append(out, "", empty)
			if i+1 == len(lines) {
				out = append(out, "")
			}
		}
	}
	return strings.Join(out, "\n")
}
