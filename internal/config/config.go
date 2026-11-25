package config

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
	"go.yaml.in/yaml/v4"
)

type Rule struct {
	Required     bool     `yaml:"required"`
	Format       string   `yaml:"format"`
	Values       []string `yaml:"values"`
	NonEmpty     bool     `yaml:"non_empty"`
	EmptyAllowed bool     `yaml:"empty_allowed"`
}

type Kind struct {
	Folder      string            `yaml:"folder"`
	Name        string            `yaml:"name"`
	Skeleton    string            `yaml:"skeleton"`
	Set         map[string]string `yaml:"set"`
	Frontmatter map[string]Rule   `yaml:"frontmatter"`
}

type Type struct {
	Folders     []string        `yaml:"folders"`
	Frontmatter map[string]Rule `yaml:"frontmatter"`
	Kinds       map[string]Kind `yaml:"kinds"`
}

type Groups struct {
	ByFolder bool
	Headings map[string]string
}

type Index struct {
	File       string
	Check      bool
	Include    []string
	OtherLinks string
	Groups     Groups
	Sort       string
}

type Entry struct {
	Labels struct {
		Created  string `yaml:"created"`
		Modified string `yaml:"modified"`
		Note     string `yaml:"note"`
	} `yaml:"labels"`
	Separator string `yaml:"separator"`
	OneLine   bool   `yaml:"one_line"`
	AsLinks   bool   `yaml:"as_links"`
}

type Log struct {
	Mode       string   `yaml:"mode"`
	Path       string   `yaml:"path"`
	Operations []string `yaml:"operations"`
	Entry      Entry    `yaml:"entry"`
}

type Frozen struct {
	Path    string
	Checked bool
	Except  []string
}

type Citations struct {
	Frontmatter string `yaml:"frontmatter"`
	Code        bool   `yaml:"code"`
}

type Orphans struct {
	Scope                []string `yaml:"scope"`
	IgnoreLinksFrom      []string `yaml:"ignore_links_from"`
	IgnoreLinksFromTypes []string `yaml:"ignore_links_from_types"`
	ExemptTypes          []string `yaml:"exempt_types"`
}

type Manifest struct {
	File     string   `yaml:"file"`
	Pipeline string   `yaml:"pipeline"`
	Statuses []string `yaml:"statuses"`
}

type Config struct {
	Version              int             `yaml:"-"`
	Root                 string          `yaml:"root"`
	Locale               string          `yaml:"locale"`
	Title                string          `yaml:"title"`
	Links                string          `yaml:"links"`
	Ignore               []string        `yaml:"ignore"`
	UncheckedFrontmatter []string        `yaml:"unchecked_frontmatter"`
	Frontmatter          map[string]Rule `yaml:"frontmatter"`
	Types                map[string]Type `yaml:"types"`
	Stray                bool            `yaml:"-"`
	Skeletons            string          `yaml:"skeletons"`
	Index                *Index          `yaml:"-"`
	Log                  *Log            `yaml:"log"`
	Frozen               []Frozen        `yaml:"-"`
	AppendOnly           []string        `yaml:"append_only"`
	BaseBranch           string          `yaml:"base_branch"`
	Citations            *Citations      `yaml:"citations"`
	Orphans              *Orphans        `yaml:"orphans"`
	TemplateMarkers      []string        `yaml:"template_markers"`
	Manifest             *Manifest       `yaml:"manifest"`
	Protected            []string        `yaml:"protected"`
}

type plainConfig Config

type rawIndex struct {
	File       string    `yaml:"file"`
	Check      *bool     `yaml:"check"`
	Include    []string  `yaml:"include"`
	OtherLinks string    `yaml:"other_links"`
	Groups     yaml.Node `yaml:"groups"`
	Sort       string    `yaml:"sort"`
}

type rawFrozen struct {
	Path    string   `yaml:"path"`
	Checked *bool    `yaml:"checked"`
	Except  []string `yaml:"except"`
}

type rawConfig struct {
	plainConfig `yaml:",inline"`
	Version     yaml.Node   `yaml:"version"`
	Stray       *bool       `yaml:"stray"`
	Index       *rawIndex   `yaml:"index"`
	Frozen      []rawFrozen `yaml:"frozen"`
}

func Load(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, exit.Usage("%v", err)
	}
	var head struct {
		Version yaml.Node `yaml:"version"`
	}
	if err := yaml.Load(data, &head); err != nil {
		return nil, exit.Usage("%s: %v", file, err)
	}
	if head.Version.Kind != yaml.ScalarNode || head.Version.Value != "1" {
		found := "none"
		if head.Version.Kind == yaml.ScalarNode {
			found = head.Version.Value
		}
		return nil, exit.Usage("%s: wikai expects version 1, found %s", file, found)
	}
	var raw rawConfig
	if err := yaml.Load(data, &raw, yaml.WithKnownFields(), yaml.WithUniqueKeys(true)); err != nil {
		return nil, exit.Usage("%s: %v", file, err)
	}
	c, err := build(&raw)
	if err != nil {
		return nil, exit.Usage("%s: %v", file, err)
	}
	return c, nil
}

func build(raw *rawConfig) (*Config, error) {
	c := Config(raw.plainConfig)
	c.Version = 1
	c.Stray = raw.Stray == nil || *raw.Stray
	setDefault(&c.Root, ".")
	setDefault(&c.Locale, "en")
	setDefault(&c.Title, "heading")
	setDefault(&c.Links, "markdown")
	setDefault(&c.Skeletons, "_skeletons")
	setDefault(&c.BaseBranch, "main")

	if err := oneOf("locale", c.Locale, "en", "fr"); err != nil {
		return nil, err
	}
	if err := oneOf("title", c.Title, "heading", "filename"); err != nil {
		return nil, err
	}
	if err := oneOf("links", c.Links, "markdown", "wikilink"); err != nil {
		return nil, err
	}
	if err := checkPath("root", c.Root); err != nil {
		return nil, err
	}
	c.Root = path.Clean(c.Root)
	if err := checkRules("frontmatter", c.Frontmatter); err != nil {
		return nil, err
	}
	if err := c.buildTypes(); err != nil {
		return nil, err
	}
	if err := c.buildIndex(raw.Index); err != nil {
		return nil, err
	}
	if err := c.buildLog(); err != nil {
		return nil, err
	}
	for _, f := range raw.Frozen {
		if err := checkPath("frozen.path", f.Path); err != nil {
			return nil, err
		}
		if err := checkPaths("frozen.except", f.Except); err != nil {
			return nil, err
		}
		c.Frozen = append(c.Frozen, Frozen{Path: f.Path, Checked: f.Checked == nil || *f.Checked, Except: f.Except})
	}
	if c.Orphans != nil && len(c.Orphans.Scope) == 0 {
		c.Orphans.Scope = c.typeFolders()
	}
	if c.Orphans != nil {
		if err := checkPaths("orphans.scope", c.Orphans.Scope); err != nil {
			return nil, err
		}
	}
	if c.Manifest != nil {
		if c.Manifest.File == "" {
			return nil, fmt.Errorf("manifest.file is required")
		}
		if err := checkPath("manifest.file", c.Manifest.File); err != nil {
			return nil, err
		}
	}
	for key, list := range map[string][]string{
		"ignore": c.Ignore, "unchecked_frontmatter": c.UncheckedFrontmatter,
		"append_only": c.AppendOnly, "protected": c.Protected,
	} {
		if err := checkPaths(key, list); err != nil {
			return nil, err
		}
	}
	if err := checkPath("skeletons", c.Skeletons); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) buildTypes() error {
	owner := map[string]string{}
	for _, name := range slices.Sorted(mapKeys(c.Types)) {
		t := c.Types[name]
		if err := checkPaths(fmt.Sprintf("types.%s.folders", name), t.Folders); err != nil {
			return err
		}
		if err := checkRules(fmt.Sprintf("types.%s.frontmatter", name), t.Frontmatter); err != nil {
			return err
		}
		if len(t.Kinds) == 0 {
			t.Kinds = map[string]Kind{name: {}}
		}
		for _, kn := range slices.Sorted(mapKeys(t.Kinds)) {
			k := t.Kinds[kn]
			if prev, dup := owner[kn]; dup {
				return fmt.Errorf("kind %s is declared in types %s and %s", kn, prev, name)
			}
			owner[kn] = name
			if k.Folder != "" {
				if err := checkPath(fmt.Sprintf("types.%s.kinds.%s.folder", name, kn), k.Folder); err != nil {
					return err
				}
			}
			if err := checkRules(fmt.Sprintf("types.%s.kinds.%s.frontmatter", name, kn), k.Frontmatter); err != nil {
				return err
			}
			if k.Folder == "" && len(t.Folders) > 0 {
				k.Folder = t.Folders[0]
			}
			setDefault(&k.Name, "{slug}")
			t.Kinds[kn] = k
		}
		c.Types[name] = t
	}
	return nil
}

func (c *Config) buildIndex(raw *rawIndex) error {
	if raw == nil {
		return nil
	}
	if raw.File == "" {
		return fmt.Errorf("index.file is required")
	}
	if err := checkPath("index.file", raw.File); err != nil {
		return err
	}
	idx := &Index{File: raw.File, Check: raw.Check == nil || *raw.Check, Include: raw.Include,
		OtherLinks: raw.OtherLinks, Sort: raw.Sort}
	setDefault(&idx.OtherLinks, "forbid")
	setDefault(&idx.Sort, "path")
	if err := oneOf("index.other_links", idx.OtherLinks, "forbid", "allow"); err != nil {
		return err
	}
	switch raw.Groups.Kind {
	case 0:
	case yaml.ScalarNode:
		if raw.Groups.Value != "folder" {
			return fmt.Errorf("index.groups must be \"folder\" or a map of headings, found %q", raw.Groups.Value)
		}
		idx.Groups.ByFolder = true
	case yaml.MappingNode:
		if err := raw.Groups.Decode(&idx.Groups.Headings); err != nil {
			return fmt.Errorf("index.groups: %v", err)
		}
	default:
		return fmt.Errorf("index.groups must be \"folder\" or a map of headings")
	}
	if len(idx.Include) == 0 {
		idx.Include = c.typeFolders()
	}
	if err := checkPaths("index.include", idx.Include); err != nil {
		return err
	}
	c.Index = idx
	return nil
}

func (c *Config) buildLog() error {
	l := c.Log
	if l == nil {
		return nil
	}
	setDefault(&l.Mode, "dir")
	if err := oneOf("log.mode", l.Mode, "dir", "file"); err != nil {
		return err
	}
	if l.Path == "" {
		return fmt.Errorf("log.path is required")
	}
	if err := checkPath("log.path", l.Path); err != nil {
		return err
	}
	setDefault(&l.Entry.Separator, ": ")
	setDefault(&l.Entry.Labels.Created, "created")
	setDefault(&l.Entry.Labels.Modified, "modified")
	setDefault(&l.Entry.Labels.Note, "note")
	return nil
}

func (c *Config) typeFolders() []string {
	var out []string
	for _, name := range slices.Sorted(mapKeys(c.Types)) {
		for _, f := range c.Types[name].Folders {
			if !slices.Contains(out, f) {
				out = append(out, f)
			}
		}
	}
	return out
}

func (c *Config) RootPath(p string) string {
	return path.Clean(path.Join(c.Root, p))
}

func (c *Config) KindOf(name string) (string, Kind, bool) {
	for typeName, t := range c.Types {
		if k, ok := t.Kinds[name]; ok {
			return typeName, k, true
		}
	}
	return "", Kind{}, false
}

func setDefault(s *string, def string) {
	if *s == "" {
		*s = def
	}
}

func oneOf(key, value string, allowed ...string) error {
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("%s must be one of %s, found %q", key, strings.Join(allowed, ", "), value)
}

func checkPath(key, p string) error {
	if p == "" {
		return fmt.Errorf("%s: path must not be empty", key)
	}
	if path.IsAbs(p) || strings.Contains(p, `\`) || slices.Contains(strings.Split(p, "/"), "..") {
		return fmt.Errorf("%s: %q must be relative and hold no \"..\"", key, p)
	}
	return nil
}

func checkPaths(key string, list []string) error {
	for _, p := range list {
		if err := checkPath(key, p); err != nil {
			return err
		}
	}
	return nil
}

func checkRules(key string, rules map[string]Rule) error {
	for _, name := range slices.Sorted(mapKeys(rules)) {
		if err := oneOf(fmt.Sprintf("%s.%s.format", key, name), rules[name].Format, "", "date", "list"); err != nil {
			return err
		}
	}
	return nil
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
