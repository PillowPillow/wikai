package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
)

type Change struct {
	Status, Path string
}

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", args[0], msg)
	}
	return stdout.Bytes(), nil
}

func splitZ(out []byte) []string {
	s := strings.TrimSuffix(string(out), "\x00")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x00")
}

func IsRepo(dir string) bool {
	out, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func HasCommit(dir string) bool {
	_, err := run(dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	return err == nil
}

func LsFiles(dir string) ([]string, error) {
	out, err := run(dir, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	return splitZ(out), nil
}

func LsTree(dir, ref string, paths ...string) ([]string, error) {
	out, err := run(dir, append([]string{"ls-tree", "-r", "-z", "--name-only", ref, "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	return splitZ(out), nil
}

// The "./" prefix resolves the path from dir instead of the repository top level.
func Show(dir, ref, path string) ([]byte, error) {
	return run(dir, "cat-file", "blob", ref+":./"+path)
}

func MergeBase(dir, a, b string) (string, error) {
	out, err := run(dir, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func Diff(dir string, args ...string) ([]Change, error) {
	out, err := run(dir, append([]string{"diff", "--name-status", "--no-renames", "--relative", "-z"}, args...)...)
	if err != nil {
		return nil, err
	}
	fields := splitZ(out)
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("git diff: unexpected output")
	}
	var changes []Change
	for i := 0; i < len(fields); i += 2 {
		changes = append(changes, Change{Status: fields[i], Path: fields[i+1]})
	}
	return changes, nil
}

func Mv(dir, from, to string) error {
	_, err := run(dir, "mv", "--", from, to)
	return err
}

func BaseRef(dir, baseBranch, flag string) (ref string, warnings []string, err error) {
	if !IsRepo(dir) {
		return "", []string{"not a git repository, frozen check skipped"}, nil
	}
	if !HasCommit(dir) {
		return "", nil, nil
	}
	if flag != "" {
		return flag, nil, nil
	}
	out, err := run(dir, "remote")
	if err != nil {
		return "", nil, exit.Usage("%v", err)
	}
	if !slices.Contains(strings.Split(strings.TrimSpace(string(out)), "\n"), "origin") {
		return "HEAD", []string{"no remote, base is HEAD"}, nil
	}
	base, err := MergeBase(dir, "HEAD", "origin/"+baseBranch)
	if err != nil {
		return "", nil, exit.Usage("no merge-base with origin/%s; fetch origin or pass --base", baseBranch)
	}
	return base, nil, nil
}
