package check

import (
	"slices"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/wiki"
)

func TestRunSortsByPathThenCheck(t *testing.T) {
	b := &wiki.Base{
		Boundary:     t.TempDir(),
		Cfg:          &config.Config{BaseBranch: "main"},
		LoadFindings: []wiki.Finding{{Check: "frontmatter", Path: "c.md", Message: "not UTF-8"}},
	}
	checks := []Check{
		{Name: "zeta", Run: func(*wiki.Base) []wiki.Finding {
			return []wiki.Finding{{Check: "zeta", Path: "a.md", Message: "z"}}
		}},
		{Name: "alpha", Run: func(*wiki.Base) []wiki.Finding {
			return []wiki.Finding{{Check: "alpha", Path: "b.md", Message: "1"}, {Check: "alpha", Path: "a.md", Message: "2"}}
		}},
	}
	got, _, err := run(b, checks, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []wiki.Finding{
		{Check: "alpha", Path: "a.md", Message: "2"},
		{Check: "zeta", Path: "a.md", Message: "z"},
		{Check: "alpha", Path: "b.md", Message: "1"},
		{Check: "frontmatter", Path: "c.md", Message: "not UTF-8"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestRunReturnsEmptyNotNil(t *testing.T) {
	b := &wiki.Base{Boundary: t.TempDir(), Cfg: &config.Config{BaseBranch: "main"}}
	got, warnings, err := run(b, nil, "")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	if !slices.Contains(warnings, "not a git repository, frozen check skipped") {
		t.Fatalf("warnings=%q", warnings)
	}
}
