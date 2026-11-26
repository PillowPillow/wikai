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

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/wiki"
)

type LogOpts struct {
	Operation, Title, Slug   string
	Created, Modified, Notes []string
	Date                     time.Time
}

func NewLog(b *wiki.Base, o LogOpts) (string, error) {
	l := b.Cfg.Log
	if l == nil {
		return "", exit.Usage("this base has no log block")
	}
	if !slices.Contains(l.Operations, o.Operation) {
		return "", exit.Usage("operation %q is not one of: %s", o.Operation, strings.Join(l.Operations, ", "))
	}
	title, err := oneLine("the title", strings.TrimSpace(o.Title))
	if err != nil {
		return "", err
	}
	if title == "" {
		return "", exit.Usage("the title is empty")
	}
	if len(o.Created)+len(o.Modified)+len(o.Notes) == 0 {
		return "", exit.Usage("an entry needs at least one --created, --modified or --note")
	}
	day := o.Date
	if day.IsZero() {
		day = time.Now()
	}
	date := day.Format("2006-01-02")

	file := b.LogPath()
	if l.Mode == "dir" {
		slug := o.Slug
		if slug == "" {
			slug = Slugify(title)
		}
		if slug == "" {
			return "", exit.Usage("the title gives an empty slug: pass --slug")
		}
		name := date + "-" + o.Operation + "-" + slug
		if strings.ContainsAny(name, "/\\\x00\r\n") {
			return "", exit.Usage("the log file name %q is not a file name", name)
		}
		file = nfc(path.Join(file, name+".md"))
	}

	body, err := logBody(b, file, date, o.Operation, title, o)
	if err != nil {
		return "", err
	}
	disk := filepath.Join(b.Boundary, filepath.FromSlash(file))

	if l.Mode == "dir" {
		if err := os.MkdirAll(filepath.Dir(disk), 0o755); err != nil {
			return "", err
		}
		f, err := os.OpenFile(disk, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return "", exit.Refused("%s exists: log entries are never modified, choose another --slug", file)
		} else if err != nil {
			return "", err
		}
		_, err = f.WriteString(body)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return file, err
	}

	f, err := os.OpenFile(disk, os.O_RDWR|os.O_APPEND, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return "", exit.Usage("%s is missing", file)
	} else if err != nil {
		return "", err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return "", err
	}
	prefix := ""
	if info.Size() > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, info.Size()-1); err != nil {
			f.Close()
			return "", err
		}
		prefix = "\n"
		if last[0] != '\n' {
			prefix = "\n\n"
		}
	}
	_, err = f.WriteString(prefix + body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return file, err
}

func logBody(b *wiki.Base, file, date, operation, title string, o LogOpts) (string, error) {
	e := b.Cfg.Log.Entry
	var sb strings.Builder
	sb.WriteString("## [" + date + "] " + operation + " | " + title + "\n")
	line := func(label string, items []string) {
		if len(items) == 0 {
			return
		}
		sb.WriteString("- " + label + e.Separator + strings.Join(items, ", ") + "\n")
	}
	group := func(label string, items []string, links bool) error {
		var out []string
		for _, it := range items {
			it, err := oneLine("an entry item", it)
			if err != nil {
				return err
			}
			if links {
				it = logItem(b, file, it)
			}
			out = append(out, it)
		}
		if e.OneLine {
			line(label, out)
			return nil
		}
		for _, it := range out {
			line(label, []string{it})
		}
		return nil
	}
	if err := group(e.Labels.Created, o.Created, e.AsLinks); err != nil {
		return "", err
	}
	if err := group(e.Labels.Modified, o.Modified, e.AsLinks); err != nil {
		return "", err
	}
	for _, n := range o.Notes {
		if err := group(e.Labels.Note, []string{n}, false); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

func logItem(b *wiki.Base, file, item string) string {
	if strings.HasSuffix(item, ".md") {
		if p := b.Page(item); p != nil {
			return LinkTo(b, file, item, p.Title)
		}
		if _, err := os.Stat(filepath.Join(b.Boundary, filepath.FromSlash(item))); err == nil {
			return LinkTo(b, file, item, strings.TrimSuffix(path.Base(item), ".md"))
		}
	}
	if b.Cfg.Links == "wikilink" {
		return "[[" + item + "]]"
	}
	return item
}
