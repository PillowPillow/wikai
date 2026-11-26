package check

import (
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/wiki"
	"golang.org/x/text/unicode/norm"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type keyRule struct {
	key  string
	rule config.Rule
}

func Frontmatter(b *wiki.Base) []wiki.Finding {
	var findings []wiki.Finding
	for _, p := range b.Pages {
		if p.Unchecked {
			continue
		}
		for _, msg := range pageFrontmatter(b.Cfg, p) {
			findings = append(findings, wiki.Finding{Check: "frontmatter", Path: p.Path, Message: msg})
		}
	}
	return findings
}

func pageFrontmatter(cfg *config.Config, p *wiki.Page) []string {
	if p.MetaErr != nil {
		return []string{"invalid YAML: " + p.MetaErr.Error()}
	}
	if p.Meta == nil {
		return []string{"no frontmatter"}
	}
	var msgs []string
	layers := []map[string]config.Rule{cfg.Frontmatter}
	if msg := typeFinding(cfg, p); msg != "" {
		msgs = append(msgs, msg)
	} else if t, ok := cfg.Types[p.Type]; ok {
		layers = append(layers, t.Frontmatter)
		for _, name := range slices.Sorted(maps.Keys(t.Kinds)) {
			if k := t.Kinds[name]; kindMatches(k, p.Meta) {
				layers = append(layers, k.Frontmatter)
			}
		}
	}
	for _, r := range mergeRules(layers) {
		if msg := ruleFinding(r.key, r.rule, p.Meta); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func typeFinding(cfg *config.Config, p *wiki.Page) string {
	v, ok := p.Meta["type"]
	if !ok {
		return ""
	}
	t, declared := cfg.Types[p.Type]
	if _, isString := v.(string); !isString || !declared {
		return fmt.Sprintf("unknown type: %q", fmt.Sprint(v))
	}
	if len(t.Folders) == 0 || p.Folder == "" {
		return ""
	}
	var owners []string
	for name, other := range cfg.Types {
		if slices.ContainsFunc(other.Folders, func(f string) bool { return norm.NFC.String(path.Clean(f)) == p.Folder }) {
			owners = append(owners, name)
		}
	}
	if slices.Contains(owners, p.Type) {
		return ""
	}
	slices.Sort(owners)
	return fmt.Sprintf("type must be %s in %s/, found %q", strings.Join(owners, " or "), p.Folder, p.Type)
}

func kindMatches(k config.Kind, meta map[string]any) bool {
	for key, want := range k.Set {
		if got, ok := meta[key].(string); !ok || got != want {
			return false
		}
	}
	return true
}

func mergeRules(layers []map[string]config.Rule) []keyRule {
	var out []keyRule
	for _, layer := range layers {
		for _, key := range slices.Sorted(maps.Keys(layer)) {
			i := slices.IndexFunc(out, func(r keyRule) bool { return r.key == key })
			if i >= 0 {
				out[i].rule = layer[key]
			} else {
				out = append(out, keyRule{key, layer[key]})
			}
		}
	}
	return out
}

func ruleFinding(key string, r config.Rule, meta map[string]any) string {
	v, ok := meta[key]
	if !ok {
		if r.Required {
			return "missing key: " + key
		}
		return ""
	}
	empty := v == ""
	skipEmpty := empty && r.EmptyAllowed
	switch {
	case r.Format == "date" && !skipEmpty && !validDate(v):
		return key + " is not YYYY-MM-DD"
	case r.Format == "list" && !isList(v):
		return key + " is not a list"
	case len(r.Values) > 0 && !skipEmpty && !slices.Contains(r.Values, fmt.Sprint(v)):
		return fmt.Sprintf("%s must be one of %s, found %q", key, strings.Join(r.Values, ", "), fmt.Sprint(v))
	case r.NonEmpty && (empty || listLen(v) == 0):
		return key + " is empty"
	}
	return ""
}

func validDate(v any) bool {
	s, ok := v.(string)
	if !ok || !datePattern.MatchString(s) {
		return false
	}
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

func isList(v any) bool {
	switch v.(type) {
	case []string, []any:
		return true
	}
	return false
}

func listLen(v any) int {
	switch l := v.(type) {
	case []string:
		return len(l)
	case []any:
		return len(l)
	}
	return -1
}
