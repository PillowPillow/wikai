package check

import (
	"cmp"
	"slices"

	"github.com/PillowPillow/wikai/internal/gitx"
	"github.com/PillowPillow/wikai/internal/wiki"
)

type Check struct {
	Name string
	Run  func(*wiki.Base) []wiki.Finding
}

func All() []Check {
	return []Check{
		{"stray", Stray},
		{"frontmatter", Frontmatter},
		{"links", Links},
		{"index", Index},
		{"duplicates", Duplicates},
		{"orphans", Orphans},
	}
}

func Run(b *wiki.Base, baseFlag string) (findings []wiki.Finding, warnings []string, err error) {
	return run(b, All(), baseFlag)
}

func run(b *wiki.Base, checks []Check, baseFlag string) ([]wiki.Finding, []string, error) {
	_, warnings, err := gitx.BaseRef(b.Boundary, b.Cfg.BaseBranch, baseFlag)
	if err != nil {
		return nil, warnings, err
	}
	findings := append([]wiki.Finding{}, b.LoadFindings...)
	for _, c := range checks {
		findings = append(findings, c.Run(b)...)
	}
	slices.SortStableFunc(findings, func(x, y wiki.Finding) int {
		return cmp.Or(cmp.Compare(x.Path, y.Path), cmp.Compare(x.Check, y.Check))
	})
	return findings, warnings, nil
}
