package check

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/wiki"
)

func TestTemplatesReportsAnUnreadableSkeleton(t *testing.T) {
	dir := t.TempDir()
	skel := filepath.Join(dir, "_skeletons", "a.md")
	if err := os.MkdirAll(filepath.Dir(skel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skel, []byte("{{date:Q}}"), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(skel); err == nil {
		t.Skip("file mode does not block reading here")
	}
	b := &wiki.Base{Boundary: dir, Cfg: &config.Config{Root: ".", Skeletons: "_skeletons"}}
	got := Templates(b)
	want := []wiki.Finding{{Check: "templates", Path: "_skeletons/a.md", Message: "not readable: permission denied"}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestTemplatesIgnoresAMissingSkeletonFolder(t *testing.T) {
	b := &wiki.Base{Boundary: t.TempDir(), Cfg: &config.Config{Root: ".", Skeletons: "_skeletons"}}
	if got := Templates(b); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}
