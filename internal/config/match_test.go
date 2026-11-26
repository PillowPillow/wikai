package config

import "testing"

func TestUnder(t *testing.T) {
	for _, c := range []struct {
		p, dir string
		want   bool
	}{
		{"raw/a.md", "raw", true},
		{"raw/a.md", "RAW", true},
		{"Raw", "raw/", true},
		{"rawx/a.md", "raw", false},
		{"raw", "raw/a.md", false},
		{"a/b.md", ".", true},
		{"./raw/../raw/a.md", "raw", true},
		{"Idée/x.md", "Idée", true},
	} {
		if got := Under(c.p, c.dir); got != c.want {
			t.Errorf("Under(%q, %q) = %v, want %v", c.p, c.dir, got, c.want)
		}
	}
}

func TestSamePath(t *testing.T) {
	if !SamePath("_meta/manifest.jsonl", "_META/Manifest.jsonl") || SamePath("_meta/manifest.jsonl/x", "_meta/manifest.jsonl") || SamePath("_meta", "_meta/manifest.jsonl") {
		t.Fatal("SamePath")
	}
}
