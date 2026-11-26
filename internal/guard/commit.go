package guard

import (
	"bytes"
	"sort"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/gitx"
	"golang.org/x/text/unicode/norm"
)

type Violation struct {
	Status, Path, Reason string
}

func Commit(cfg *config.Config, boundary string, staged bool, base, head string, agent bool) ([]Violation, error) {
	oldRef, newRef := "HEAD", ""
	var changes []gitx.Change
	var err error
	if staged {
		changes, err = gitx.Diff(boundary, "--cached")
	} else {
		if oldRef, err = gitx.MergeBase(boundary, base, head); err != nil {
			return nil, err
		}
		newRef = head
		changes, err = gitx.Diff(boundary, base+"..."+head)
	}
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, c := range changes {
		p := norm.NFC.String(c.Path)
		reason := ""
		switch Classify(cfg, p) {
		case Frozen:
			if c.Status != "A" {
				reason = "frozen file"
			}
		case AppendOnly:
			if reason, err = appendOnly(boundary, c, oldRef, newRef); err != nil {
				return nil, err
			}
		case Protected:
			if agent {
				reason = "protected path"
			}
		}
		if reason != "" {
			out = append(out, Violation{Status: c.Status, Path: p, Reason: reason})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// An empty newRef reads the new side from the index.
func appendOnly(dir string, c gitx.Change, oldRef, newRef string) (string, error) {
	const reason = "append-only file: existing lines changed"
	if c.Status == "A" {
		return "", nil
	}
	if c.Status == "D" {
		return reason, nil
	}
	old, err := gitx.Show(dir, oldRef, c.Path)
	if err != nil {
		return "", err
	}
	cur, err := gitx.Show(dir, newRef, c.Path)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(cur, old) {
		return reason, nil
	}
	return "", nil
}
