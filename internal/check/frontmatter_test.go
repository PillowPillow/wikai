package check

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/wiki"
)

const rulesConfig = `version: 1
root: .
links: wikilink
stray: false
frontmatter:
  type:    { required: true }
  created: { required: true, format: date }
  sources: { required: true, format: list }
types:
  wiki:
    folders: [20-wiki, 50-missions]
    frontmatter:
      status:   { required: true, values: [draft, done] }
      sources:  { required: true, format: list, non_empty: true }
      reviewed: { required: true, format: date, empty_allowed: true }
  mission:
    folders: [50-missions]
  note:
    folders: [30-notes]
    frontmatter:
      topic: { required: true }
  meta: {}
`

func frontmatterOf(t *testing.T, folder, text string) []string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "wikai.yaml")
	if err := os.WriteFile(file, []byte(rulesConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(file)
	if err != nil {
		t.Fatal(err)
	}
	meta, _, _, metaErr := wiki.ParseFrontmatter(text)
	page := &wiki.Page{Path: folder + "/p.md", Meta: meta, MetaErr: metaErr, Folder: folder}
	page.Type, _ = meta["type"].(string)
	var messages []string
	for _, f := range Frontmatter(&wiki.Base{Cfg: cfg, Pages: []*wiki.Page{page}}) {
		messages = append(messages, f.Message)
	}
	return messages
}

func TestFrontmatterRules(t *testing.T) {
	const wikiHead = "---\ntype: wiki\ncreated: 2026-01-01\nsources: [s]\nstatus: done\nreviewed:\n"
	cases := []struct {
		name, folder, text string
		want               []string
	}{
		{"valid wiki page", "20-wiki", wikiHead + "---\n", nil},
		{"not a mapping", "20-wiki", "---\n- a\n---\n", []string{"invalid YAML: not a mapping"}},
		{"empty frontmatter", "20-wiki", "---\n---\n", []string{"missing key: created", "missing key: sources", "missing key: type"}},
		{"unknown type gets top-level rules only", "20-wiki", "---\ntype: nope\ncreated: x\nsources: []\n---\n",
			[]string{`unknown type: "nope"`, "created is not YYYY-MM-DD"}},
		{"type of another folder", "50-missions", "---\ntype: note\ncreated: 2026-01-01\nsources: []\n---\n",
			[]string{`type must be mission or wiki in 50-missions/, found "note"`}},
		{"type without folders sits anywhere", "20-wiki", "---\ntype: meta\ncreated: 2026-01-01\nsources: []\n---\n", nil},
		{"type rule replaces top-level rule of the same key", "20-wiki",
			"---\ntype: wiki\ncreated: 2026-01-01\nstatus: done\nreviewed:\n---\n", []string{"missing key: sources"}},
		{"empty list", "20-wiki", "---\ntype: wiki\ncreated: 2026-01-01\nsources: []\nstatus: done\nreviewed:\n---\n",
			[]string{"sources is empty"}},
		{"not a list", "20-wiki", "---\ntype: wiki\ncreated: 2026-01-01\nsources: s\nstatus: done\nreviewed:\n---\n",
			[]string{"sources is not a list"}},
		{"empty value without empty_allowed", "20-wiki", "---\ntype: wiki\ncreated:\nsources: [s]\nstatus:\nreviewed:\n---\n",
			[]string{"created is not YYYY-MM-DD", `status must be one of draft, done, found ""`}},
		{"impossible date", "20-wiki", "---\ntype: wiki\ncreated: 2026-01-01\nsources: [s]\nstatus: done\nreviewed: 2026-13-01\n---\n",
			[]string{"reviewed is not YYYY-MM-DD"}},
		{"duplicate key", "20-wiki", "---\ntype: wiki\ntype: wiki\n---\n", []string{`invalid YAML: duplicate key "type"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := frontmatterOf(t, c.folder, c.text)
			slices.Sort(got)
			want := slices.Clone(c.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}
