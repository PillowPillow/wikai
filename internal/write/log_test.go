package write

import (
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/testrepo"
	"github.com/PillowPillow/wikai/internal/wiki"
)

func TestNewLogDir(t *testing.T) {
	b, dir := homeBase(t)
	o := LogOpts{Operation: "ingest", Title: "Billing notes", Created: []string{"knowledge/glossary/offer.md"}, Modified: []string{"knowledge/index.md"}, Notes: []string{"first", "second"}, Date: jan2}
	p, err := NewLog(b, o)
	if err != nil || p != "knowledge/log/2026-01-02-ingest-billing-notes.md" {
		t.Fatalf("%q %v", p, err)
	}
	want := "## [2026-01-02] ingest | Billing notes\n- created: knowledge/glossary/offer.md\n- modified: knowledge/index.md\n- note: first\n- note: second\n"
	if got := read(t, dir, p); got != want {
		t.Fatalf("content = %q", got)
	}
	o.Notes = []string{"other"}
	if _, err := NewLog(b, o); exit.Code(err) != 1 || !strings.Contains(err.Error(), "log entries are never modified, choose another --slug") {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, dir, p); got != want {
		t.Fatalf("content changed: %q", got)
	}
	if _, err := NewLog(b, LogOpts{Operation: "deploy", Title: "x", Notes: []string{"n"}, Date: jan2}); exit.Code(err) != 2 {
		t.Fatalf("err = %v", err)
	}
	if _, err := NewLog(b, LogOpts{Operation: "ingest", Title: "x", Date: jan2}); exit.Code(err) != 2 {
		t.Fatalf("err = %v", err)
	}
	if _, err := NewLog(b, LogOpts{Operation: "ingest", Title: "!!!", Notes: []string{"n"}, Date: jan2}); exit.Code(err) != 2 {
		t.Fatalf("err = %v", err)
	}
	if p, err := NewLog(b, LogOpts{Operation: "ingest", Title: "!!!", Slug: "s", Notes: []string{"n"}, Date: jan2}); err != nil || p != "knowledge/log/2026-01-02-ingest-s.md" {
		t.Fatalf("%q %v", p, err)
	}
}

func TestNewLogFile(t *testing.T) {
	_, dir := vaultBase(t)
	const before = "# Log\n\n## [2025-12-01] ingest | Old\n- note : old\n"
	testrepo.Write(t, dir, map[string]string{
		"log.md":                           before,
		"20-wiki/concepts/Evals.md":        "---\ntype: wiki\n---\n",
		"20-wiki/resumes/Résumé - Talk.md": "---\ntype: wiki\n---\n",
	})
	b, _ := loadBase(t, dir)
	p, err := NewLog(b, LogOpts{Operation: "ingest", Title: "Talk", Created: []string{"20-wiki/resumes/Résumé - Talk.md", "20-wiki/concepts/Evals.md"}, Modified: []string{"index.md"}, Notes: []string{"done"}, Date: jan2})
	if err != nil || p != "log.md" {
		t.Fatalf("%q %v", p, err)
	}
	want := before + "\n## [2026-01-02] ingest | Talk\n- créé : [[Résumé - Talk]], [[Evals]]\n- modifié : [[index]]\n- note : done\n"
	if got := read(t, dir, p); got != want {
		t.Fatalf("content = %q", got)
	}
}

func TestNewLogFileMissing(t *testing.T) {
	b, _ := vaultBase(t)
	if _, err := NewLog(b, LogOpts{Operation: "ingest", Title: "T", Notes: []string{"n"}, Date: jan2}); exit.Code(err) != 2 {
		t.Fatalf("err = %v", err)
	}
}

func TestNewLogNoBlock(t *testing.T) {
	dir := testrepo.New(t, map[string]string{"wikai.yaml": "version: 1\nroot: kb\ntypes:\n  note:\n    folders: [notes]\n", "kb/notes/a.md": "---\ntype: note\n---\n"})
	b, _ := loadBase(t, dir)
	_, err := NewLog(b, LogOpts{Operation: "ingest", Title: "T", Notes: []string{"n"}, Date: jan2})
	if exit.Code(err) != 2 {
		t.Fatalf("err = %v", err)
	}
}

func TestNewLogFileNoTrailingNewline(t *testing.T) {
	_, dir := vaultBase(t)
	testrepo.Write(t, dir, map[string]string{"log.md": "x"})
	var b *wiki.Base
	b, _ = loadBase(t, dir)
	if _, err := NewLog(b, LogOpts{Operation: "query", Title: "T", Notes: []string{"a", "b"}, Date: jan2}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "log.md"); got != "x\n\n## [2026-01-02] query | T\n- note : a\n- note : b\n" {
		t.Fatalf("content = %q", got)
	}
}
