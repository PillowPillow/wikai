package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func runHook(t *testing.T, dir, stdin string, env map[string]string) (code int, out, errOut string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code = Execute([]string{"guard", "hook"}, Env{
		Dir:    dir,
		Stdin:  strings.NewReader(stdin),
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(k string) string { return env[k] },
		Now:    func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	return code, stdout.String(), stderr.String()
}

func guardVault(t *testing.T) string {
	t.Helper()
	cfg, err := os.ReadFile(filepath.Join("..", "config", "testdata", "brain.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return testrepo.New(t, map[string]string{
		"wikai.yaml":            string(cfg),
		"CLAUDE.md":             "# rules\n",
		"10-raw/web/a.md":       "a\n",
		"_meta/manifest.jsonl":  "L1\n",
		"20-wiki/concepts/A.md": "A\n",
	})
}

func editEvent(file, cwd string) string {
	return fmt.Sprintf(`{"session_id":"s","hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":%q},"cwd":%q}`, file, cwd)
}

func TestHookCLI(t *testing.T) {
	vault := guardVault(t)
	elsewhere := t.TempDir()
	broken := t.TempDir()
	testrepo.Write(t, broken, map[string]string{"wikai.yaml": "colour: blue\n"})
	claude := filepath.Join(vault, "CLAUDE.md")
	cases := []struct {
		name, dir, stdin string
		env              map[string]string
		code             int
		errOut           string
	}{
		{"protected", vault, editEvent(claude, vault), nil, 2, "CLAUDE.md: protected path, propose the change to the human\n"},
		{"case variant of a missing protected file", vault, editEvent(filepath.Join(vault, "Agents.MD"), vault), nil, 2, "Agents.MD: protected path, propose the change to the human\n"},
		{"allowed", vault, editEvent(filepath.Join(vault, "20-wiki/concepts/A.md"), vault), nil, 0, ""},
		{"not json", vault, "not json", nil, 2, "error: "},
		{"json null", vault, "null", nil, 2, "error: "},
		{"json array", vault, "[]", nil, 2, "error: "},
		{"trailing garbage", vault, editEvent(claude, vault) + "x", nil, 2, "error: "},
		{"wrong field type", vault, `{"tool_input":{"file_path":3}}`, nil, 2, "error: "},
		{"no wikai.yaml", elsewhere, editEvent(filepath.Join(elsewhere, "CLAUDE.md"), elsewhere), nil, 0, ""},
		{"invalid wikai.yaml", broken, editEvent(filepath.Join(broken, "x.md"), broken), nil, 2, "error: "},
		{"project dir", elsewhere, editEvent(claude, elsewhere), map[string]string{"CLAUDE_PROJECT_DIR": vault}, 2, "CLAUDE.md: protected path, propose the change to the human\n"},
		{"event cwd", elsewhere, editEvent(claude, vault), nil, 2, "CLAUDE.md: protected"},
		{"no cwd uses the start folder", vault, `{"tool_name":"Edit","tool_input":{"file_path":"CLAUDE.md"}}`, nil, 2, "CLAUDE.md: protected"},
		{"no target", vault, `{"tool_name":"Edit","tool_input":{},"cwd":"` + vault + `"}`, nil, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errOut := runHook(t, c.dir, c.stdin, c.env)
			exact := strings.HasSuffix(c.errOut, "\n") && errOut != c.errOut
			if code != c.code || out != "" || exact || !strings.HasPrefix(errOut, c.errOut) || (c.errOut == "") != (errOut == "") {
				t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
			}
		})
	}
}

func TestCommitCLI(t *testing.T) {
	vault := guardVault(t)
	code, out, errOut := run(t, vault, "guard", "commit", "--staged")
	if code != 0 || out != "guard: OK\n" || errOut != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	testrepo.Write(t, vault, map[string]string{"10-raw/web/a.md": "a2\n", "_meta/manifest.jsonl": "X\n"})
	testrepo.Git(t, vault, "add", "-A")
	code, out, errOut = run(t, filepath.Join(vault, "20-wiki"), "guard", "commit", "--staged")
	want := "guard: changes refused\nM 10-raw/web/a.md: frozen file\nM _meta/manifest.jsonl: append-only file: existing lines changed\n"
	if code != 1 || out != "" || errOut != want {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	testrepo.Git(t, vault, "commit", "-q", "-m", "change")
	code, _, errOut = run(t, vault, "guard", "commit", "HEAD~1", "HEAD")
	if code != 1 || errOut != want {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	code, out, _ = run(t, vault, "guard", "commit", "HEAD", "HEAD")
	if code != 0 || out != "guard: OK\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestCommitCLIUsage(t *testing.T) {
	vault := guardVault(t)
	for _, args := range [][]string{
		{"guard", "commit"}, {"guard", "commit", "--staged", "a", "b"}, {"guard", "commit", "HEAD"},
		{"guard", "commit", "HEAD", "nope"}, {"guard", "commit", "--staged", "--agent", "x"},
	} {
		code, out, errOut := run(t, vault, args...)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "error: ") {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errOut)
		}
	}
	code, _, errOut := run(t, t.TempDir(), "guard", "commit", "--staged")
	if code != 2 || !strings.Contains(errOut, "no wikai.yaml") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}
