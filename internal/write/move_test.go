package write

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/testrepo"
	"github.com/PillowPillow/wikai/internal/wiki"
)

var jan3 = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

const moveIndex = "# Index\n\n## glossary\n\n- [billing](glossary/billing.md) — billing of a customer\n- [contract](glossary/contract.md) — commercial commitment of a customer\n\n## repos\n\n- [example-backend](repos/example-backend.md) — backend repository of the acme platform\n\n## domains\n\n- [api-contract](domains/api-contract.md) — public API domain exposed to partners\n"

func homeRepo(t *testing.T, files map[string]string) (*wiki.Base, string) {
	t.Helper()
	_, dir := homeBase(t)
	all := map[string]string{
		"knowledge/index.md":            moveIndex,
		"knowledge/glossary/billing.md": "---\ntype: term\ncreated: 2026-01-01\nupdated: 2026-01-01\nsources: []\naliases: []\n---\n\n# billing\n\n```\n[contract](contract.md)\n```\n",
	}
	for k, v := range files {
		all[k] = v
	}
	testrepo.Write(t, dir, all)
	testrepo.Git(t, dir, "init", "-q", "-b", "main")
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Git(t, dir, "commit", "-q", "-m", "init")
	return loadBase(t, dir)
}

func porcelain(t *testing.T, dir string) string {
	t.Helper()
	return testrepo.Git(t, dir, "status", "--porcelain")
}

func TestMoveRename(t *testing.T) {
	b, dir := homeRepo(t, nil)
	res, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/glossary/agreement.md", Date: jan3})
	if err != nil {
		t.Fatal(err)
	}
	if exists(dir, "knowledge/glossary/contract.md") || !exists(dir, "knowledge/glossary/agreement.md") {
		t.Fatal("file not moved")
	}
	if got := read(t, dir, "knowledge/domains/api-contract.md"); !strings.Contains(got, "(../glossary/agreement.md#definition)") {
		t.Fatalf("api-contract = %q", got)
	}
	if got := read(t, dir, "knowledge/glossary/billing.md"); !strings.Contains(got, "```\n[contract](contract.md)\n```\n") {
		t.Fatalf("billing = %q", got)
	}
	if got := read(t, dir, "knowledge/index.md"); !strings.Contains(got, "## glossary\n\n- [contract](glossary/agreement.md) — commercial commitment of a customer\n- [billing](glossary/billing.md) — billing of a customer\n\n## repos") {
		t.Fatalf("index = %q", got)
	}
	if strings.Join(res.Modified, ",") != "knowledge/domains/api-contract.md" {
		t.Fatalf("modified = %q", res.Modified)
	}
	if res.Log != "knowledge/log/2026-01-03-move-agreement.md" {
		t.Fatalf("log = %q", res.Log)
	}
	wantLog := "## [2026-01-03] move | Page knowledge/glossary/contract.md renamed knowledge/glossary/agreement.md\n- created: knowledge/glossary/agreement.md\n- modified: knowledge/domains/api-contract.md\n- note: renamed from knowledge/glossary/contract.md\n"
	if got := read(t, dir, res.Log); got != wantLog {
		t.Fatalf("log = %q", got)
	}
	if st := porcelain(t, dir); !strings.Contains(st, "R  knowledge/glossary/contract.md -> knowledge/glossary/agreement.md") {
		t.Fatalf("status = %q", st)
	}
}

func TestMoveNotes(t *testing.T) {
	b, dir := homeRepo(t, nil)
	res, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/glossary/agreement.md", Notes: []string{"clearer name"}, Date: jan3})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Modified, ",") != "knowledge/domains/api-contract.md" {
		t.Fatalf("modified = %q", res.Modified)
	}
	want := "## [2026-01-03] move | Page knowledge/glossary/contract.md renamed knowledge/glossary/agreement.md\n- created: knowledge/glossary/agreement.md\n- modified: knowledge/domains/api-contract.md\n- note: renamed from knowledge/glossary/contract.md\n- note: clearer name\n"
	if got := read(t, dir, res.Log); got != want {
		t.Fatalf("log = %q", got)
	}
}

func TestMoveToArchive(t *testing.T) {
	b, dir := homeRepo(t, nil)
	res, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/archive/2026/contract.md", Date: jan3})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "knowledge/index.md"); strings.Contains(got, "[contract]") || !strings.Contains(got, "## glossary\n\n- [billing](glossary/billing.md) — billing of a customer\n\n## repos") {
		t.Fatalf("index = %q", got)
	}
	moved := read(t, dir, "knowledge/archive/2026/contract.md")
	if !strings.Contains(moved, "[api-contract](../../domains/api-contract.md)") {
		t.Fatalf("moved = %q", moved)
	}
	b2, _ := loadBase(t, dir)
	for _, l := range wiki.Links("markdown", moved) {
		if target, r := b2.Resolve("knowledge/archive/2026/contract.md", l); r != wiki.Resolved || target != "knowledge/domains/api-contract.md" {
			t.Fatalf("%+v → %q %v", l, target, r)
		}
	}
	if got := read(t, dir, "knowledge/domains/api-contract.md"); !strings.Contains(got, "(../archive/2026/contract.md#definition)") {
		t.Fatalf("api-contract = %q", got)
	}
	if res.Log != "knowledge/log/2026-01-03-move-contract.md" {
		t.Fatalf("log = %q", res.Log)
	}
}

func TestMoveToArchiveRoot(t *testing.T) {
	b, dir := homeRepo(t, nil)
	res, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/archive/contract.md", Date: jan3})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "knowledge/index.md"); strings.Contains(got, "[contract]") {
		t.Fatalf("index = %q", got)
	}
	moved := read(t, dir, "knowledge/archive/contract.md")
	b2, _ := loadBase(t, dir)
	links := wiki.Links("markdown", moved)
	if len(links) != 1 {
		t.Fatalf("links = %+v", links)
	}
	if target, r := b2.Resolve("knowledge/archive/contract.md", links[0]); r != wiki.Resolved || target != "knowledge/domains/api-contract.md" {
		t.Fatalf("%q %v", target, r)
	}
	if res.Log != "knowledge/log/2026-01-03-move-contract.md" {
		t.Fatalf("log = %q", res.Log)
	}
}

func TestMoveIntoSubfolderRecomputesOwnLinks(t *testing.T) {
	b, dir := homeRepo(t, map[string]string{"knowledge/glossary/billing.md": "---\ntype: term\ncreated: 2026-01-01\nupdated: 2026-01-01\nsources: []\naliases: []\n---\n\n# billing\n\nSee [contract](contract.md).\n"})
	if _, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/glossary/legal/contract.md", NoLog: true}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "knowledge/glossary/legal/contract.md"); !strings.Contains(got, "[api-contract](../../domains/api-contract.md)") {
		t.Fatalf("moved = %q", got)
	}
	if got := read(t, dir, "knowledge/glossary/billing.md"); !strings.Contains(got, "See [contract](legal/contract.md).") {
		t.Fatalf("billing = %q", got)
	}
	if got := read(t, dir, "knowledge/index.md"); !strings.Contains(got, "- [contract](glossary/legal/contract.md) — commercial") {
		t.Fatalf("index = %q", got)
	}
}

func TestMoveRefused(t *testing.T) {
	cases := []struct {
		name, old, new string
		code           int
		extra          map[string]string
	}{
		{"other type", "knowledge/glossary/billing.md", "knowledge/repos/billing.md", 1, nil},
		{"existing target", "knowledge/glossary/billing.md", "knowledge/glossary/contract.md", 1, nil},
		{"existing log entry", "knowledge/glossary/billing.md", "knowledge/glossary/fee.md", 1, map[string]string{"knowledge/log/2026-01-03-move-fee.md": "## old\n"}},
		{"index", "knowledge/index.md", "knowledge/glossary/index.md", 2, nil},
		{"leaves the base", "knowledge/glossary/billing.md", "../x.md", 2, nil},
		{"leaves the root", "knowledge/glossary/billing.md", "x.md", 2, nil},
		{"not markdown", "knowledge/glossary/billing.md", "knowledge/glossary/billing.txt", 2, nil},
		{"missing page", "knowledge/glossary/nope.md", "knowledge/glossary/x.md", 2, nil},
		{"frozen page", "knowledge/archive/2026-01-01-example-design.md", "knowledge/archive/x.md", 2, nil},
		{"log folder", "knowledge/glossary/billing.md", "knowledge/log/billing.md", 1, nil},
		{"empty slug", "knowledge/glossary/billing.md", "knowledge/glossary/!!!.md", 2, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, dir := homeRepo(t, c.extra)
			_, err := Move(b, MoveOpts{Old: c.old, New: c.new, Date: jan3})
			if exit.Code(err) != c.code {
				t.Fatalf("err = %v", err)
			}
			if st := porcelain(t, dir); st != "" {
				t.Fatalf("status = %q", st)
			}
		})
	}
}

func TestMoveRefusedMessages(t *testing.T) {
	b, _ := homeRepo(t, nil)
	_, err := Move(b, MoveOpts{Old: "knowledge/glossary/billing.md", New: "knowledge/repos/billing.md", Date: jan3})
	if err == nil || err.Error() != "a page moves only inside its type or to a frozen path" {
		t.Fatalf("err = %v", err)
	}
	_, err = Move(b, MoveOpts{Old: "knowledge/glossary/billing.md", New: "knowledge/glossary/contract.md", Date: jan3})
	if err == nil || err.Error() != "knowledge/glossary/contract.md exists" {
		t.Fatalf("err = %v", err)
	}
	_, err = Move(b, MoveOpts{Old: "knowledge/index.md", New: "knowledge/glossary/index.md", Date: jan3})
	if err == nil || err.Error() != "knowledge/index.md is not a page" {
		t.Fatalf("err = %v", err)
	}
}

func TestMoveNotGit(t *testing.T) {
	b, dir := homeBase(t)
	_, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/glossary/agreement.md", Date: jan3})
	if exit.Code(err) != 2 || !exists(dir, "knowledge/glossary/contract.md") {
		t.Fatalf("err = %v", err)
	}
}

func TestMoveNoLog(t *testing.T) {
	b, dir := homeRepo(t, nil)
	before, _ := os.ReadDir(filepath.Join(dir, "knowledge", "log"))
	res, err := Move(b, MoveOpts{Old: "knowledge/glossary/contract.md", New: "knowledge/glossary/agreement.md", NoLog: true, Date: jan3})
	if err != nil || res.Log != "" {
		t.Fatalf("%+v %v", res, err)
	}
	after, _ := os.ReadDir(filepath.Join(dir, "knowledge", "log"))
	if len(after) != len(before) {
		t.Fatalf("log folder changed: %d → %d", len(before), len(after))
	}
}

func vaultRepo(t *testing.T, files map[string]string) (*wiki.Base, string) {
	t.Helper()
	_, dir := vaultBase(t)
	all := map[string]string{
		"index.md":                    "# Index\n\n## Concepts et comparaisons (`20-wiki/concepts/`)\n\n- [[Evals]] — mesurer un modèle\n- [[Harness]] — banc d'essai\n\n## Résumés de sources (`20-wiki/resumes/`)\n\n- [[Résumé - b]] — x\n- [[Résumé - Z]] — x\n",
		"log.md":                      "# Log\n",
		"20-wiki/concepts/Evals.md":   "---\ntype: wiki\n---\n\nVoir [[Harness]] et [[Evals#Usage]].\n",
		"20-wiki/concepts/Harness.md": "---\ntype: wiki\nrelated: \"[[Evals]]\"\n---\n\n[[Evals]], [[Evals#^b1|les evals]], [[concepts/Evals]], `[[Evals]]`, ![[Evals]].\n",
	}
	for k, v := range files {
		all[k] = v
	}
	testrepo.Write(t, dir, all)
	testrepo.Git(t, dir, "add", "-A")
	testrepo.Git(t, dir, "commit", "-q", "-m", "pages")
	return loadBase(t, dir)
}

func TestMoveVault(t *testing.T) {
	b, dir := vaultRepo(t, nil)
	res, err := Move(b, MoveOpts{Old: "20-wiki/concepts/Evals.md", New: "20-wiki/concepts/LLM evals.md", Date: jan3})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntype: wiki\nrelated: \"[[LLM evals]]\"\n---\n\n[[LLM evals]], [[LLM evals#^b1|les evals]], [[20-wiki/concepts/LLM evals]], `[[Evals]]`, ![[LLM evals]].\n"
	if got := read(t, dir, "20-wiki/concepts/Harness.md"); got != want {
		t.Fatalf("Harness = %q", got)
	}
	if got := read(t, dir, "20-wiki/concepts/LLM evals.md"); got != "---\ntype: wiki\n---\n\nVoir [[Harness]] et [[LLM evals#Usage]].\n" {
		t.Fatalf("moved = %q", got)
	}
	if got := read(t, dir, "index.md"); !strings.Contains(got, "\n\n- [[Harness]] — banc d'essai\n- [[LLM evals]] — mesurer un modèle\n\n## Résumés") {
		t.Fatalf("index = %q", got)
	}
	if strings.Join(res.Modified, ",") != "20-wiki/concepts/Harness.md" || res.Log != "log.md" {
		t.Fatalf("%+v", res)
	}
	wantLog := "# Log\n\n## [2026-01-03] move | Page 20-wiki/concepts/Evals.md renommée 20-wiki/concepts/LLM evals.md\n- créé : [[LLM evals]]\n- modifié : [[Harness]]\n- note : renommé depuis 20-wiki/concepts/Evals.md\n"
	if got := read(t, dir, "log.md"); got != wantLog {
		t.Fatalf("log = %q", got)
	}
}

func TestMoveVaultFolderOnlyRewritesPathLinks(t *testing.T) {
	b, dir := vaultRepo(t, nil)
	if _, err := Move(b, MoveOpts{Old: "20-wiki/concepts/Evals.md", New: "20-wiki/resumes/Evals.md", NoLog: true}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "20-wiki/concepts/Harness.md"); !strings.Contains(got, "\n[[Evals]], [[Evals#^b1|les evals]], [[20-wiki/resumes/Evals]],") {
		t.Fatalf("Harness = %q", got)
	}
	if got := read(t, dir, "index.md"); !strings.Contains(got, "(`20-wiki/concepts/`)\n\n- [[Harness]] — banc d'essai\n\n## Résumés de sources (`20-wiki/resumes/`)\n\n- [[Evals]] — mesurer un modèle\n- [[Résumé - b]] — x\n") {
		t.Fatalf("index = %q", got)
	}
}

func TestMoveVaultAmbiguousStemUsesPath(t *testing.T) {
	b, dir := vaultRepo(t, map[string]string{"20-wiki/resumes/LLM evals.md": "---\ntype: wiki\n---\n"})
	if _, err := Move(b, MoveOpts{Old: "20-wiki/concepts/Evals.md", New: "20-wiki/concepts/LLM evals.md", NoLog: true}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "20-wiki/concepts/Harness.md"); !strings.Contains(got, "\n[[20-wiki/concepts/LLM evals]], [[20-wiki/concepts/LLM evals#^b1|les evals]],") {
		t.Fatalf("Harness = %q", got)
	}
}

func TestMoveVaultRefusesWikilinkCharacters(t *testing.T) {
	for _, name := range []string{"a#b", "a|b", "a[b", "a]b"} {
		b, dir := vaultRepo(t, nil)
		_, err := Move(b, MoveOpts{Old: "20-wiki/concepts/Evals.md", New: "20-wiki/concepts/" + name + ".md", Date: jan3})
		if exit.Code(err) != 2 || porcelain(t, dir) != "" {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}
