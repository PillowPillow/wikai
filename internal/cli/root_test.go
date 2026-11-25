package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, dir string, args ...string) (code int, out, errOut string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	env := Env{
		Dir:    dir,
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(string) string { return "" },
		Now:    func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	code = Execute(args, env)
	return code, stdout.String(), stderr.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, t.TempDir(), "--version")
	if code != 0 || out != "dev\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := run(t, t.TempDir(), "nope")
	if code != 2 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	code, _, errOut := run(t, t.TempDir(), "--bogus")
	if code != 2 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}
