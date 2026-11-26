package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PillowPillow/wikai/internal/exit"
)

func setupTree(t *testing.T) (root, deep string) {
	t.Helper()
	root = t.TempDir()
	deep = filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "wikai.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, deep
}

func TestFind(t *testing.T) {
	root, deep := setupTree(t)
	got, err := Find(deep)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got, filepath.Join(root, "a", "wikai.yaml"))
	_, err = Find(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no wikai.yaml") {
		t.Fatalf("got %v", err)
	}
}

func TestLocate(t *testing.T) {
	root, deep := setupTree(t)
	file := filepath.Join(root, "a", "wikai.yaml")

	cfg, boundary, err := Locate(deep, "", "")
	if err != nil || cfg == nil {
		t.Fatal(err)
	}
	eq(t, boundary, filepath.Join(root, "a"))

	_, boundary, err = Locate(deep, file, "")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, boundary, deep)

	other := filepath.Join(root, "a", "b")
	_, boundary, err = Locate(deep, file, other)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, boundary, other)

	_, boundary, err = Locate(deep, "../../wikai.yaml", "..")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, boundary, filepath.Join(root, "a", "b"))

	_, _, err = Locate(deep, "", other)
	if err == nil || !strings.Contains(err.Error(), "--dir needs --config") || exit.Code(err) != 2 {
		t.Fatalf("got %v", err)
	}

	_, _, err = Locate(deep, file, filepath.Join(root, "absent"))
	if err == nil {
		t.Fatal("expected an error for a missing --dir")
	}
}
