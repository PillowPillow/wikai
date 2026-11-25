package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/testrepo"
)

func moveHome(t *testing.T) string {
	t.Helper()
	dir := homeTree(t, "valid")
	testrepo.Write(t, dir, map[string]string{
		"knowledge/index.md": "# Index\n\n## glossary\n\n- [contract](glossary/contract.md) — commitment\n\n## repos\n\n- [example-backend](repos/example-backend.md) — backend\n\n## domains\n\n- [api-contract](domains/api-contract.md) — api\n",
	})
	testrepo.Git(t, dir, "init", "-q", "-b", "main")
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Git(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func TestMoveCLIFromSubfolder(t *testing.T) {
	dir := moveHome(t)
	code, out, errOut := run(t, filepath.Join(dir, "knowledge", "glossary"), "move", "contract.md", "agreement.md", "--note", "clearer", "--date", "2026-01-03")
	want := "moved knowledge/glossary/contract.md to knowledge/glossary/agreement.md; links rewritten in 1 page(s); index updated\ncreated knowledge/log/2026-01-03-move-agreement.md\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
	data, err := os.ReadFile(filepath.Join(dir, "knowledge/log/2026-01-03-move-agreement.md"))
	if err != nil || !strings.HasSuffix(string(data), "- note: renamed from knowledge/glossary/contract.md\n- note: clearer\n") {
		t.Fatalf("%q %v", data, err)
	}
}

func TestMoveCLINoLog(t *testing.T) {
	dir := moveHome(t)
	code, out, errOut := run(t, dir, "move", "knowledge/glossary/contract.md", "knowledge/glossary/agreement.md", "--no-log")
	if code != 0 || out != "moved knowledge/glossary/contract.md to knowledge/glossary/agreement.md; links rewritten in 1 page(s); index updated\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestMoveCLIRefused(t *testing.T) {
	for name, c := range map[string]struct {
		args []string
		code int
	}{
		"other folder":    {[]string{"contract.md", "../repos/contract.md"}, 1},
		"leaves the root": {[]string{"contract.md", "../../contract.md"}, 2},
		"leaves the base": {[]string{"contract.md", "../../../contract.md"}, 2},
		"not a page":      {[]string{"../index.md", "x.md"}, 2},
		"bad date":        {[]string{"contract.md", "agreement.md", "--date", "2026-1-3"}, 2},
		"one argument":    {[]string{"contract.md"}, 2},
	} {
		t.Run(name, func(t *testing.T) {
			dir := moveHome(t)
			code, out, errOut := run(t, filepath.Join(dir, "knowledge", "glossary"), append([]string{"move"}, c.args...)...)
			if code != c.code || out != "" || errOut == "" {
				t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
			}
			if st := testrepo.Git(t, dir, "status", "--porcelain"); st != "" {
				t.Fatalf("status = %q", st)
			}
		})
	}
}

func TestMoveCLINotGit(t *testing.T) {
	dir := homeTree(t, "valid")
	code, _, errOut := run(t, dir, "move", "knowledge/glossary/contract.md", "knowledge/glossary/agreement.md")
	if code != 2 || !strings.Contains(errOut, "is not a git repository") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}
