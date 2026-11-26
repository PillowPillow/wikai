package wiki

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	meta, raw, body, err := ParseFrontmatter("---\ntype: term\ncreated: 2026-01-01\naliases: [subscription, 'offer']\nsources:\n  - a.md\norigin:\nempty: []\n---\n# T\n")
	if err != nil {
		t.Fatal(err)
	}
	if body != "# T\n" {
		t.Errorf("body = %q", body)
	}
	if raw != "type: term\ncreated: 2026-01-01\naliases: [subscription, 'offer']\nsources:\n  - a.md\norigin:\nempty: []\n" {
		t.Errorf("raw = %q", raw)
	}
	want := map[string]any{
		"type":    "term",
		"created": "2026-01-01",
		"aliases": []string{"subscription", "offer"},
		"sources": []string{"a.md"},
		"origin":  "",
		"empty":   []string{},
	}
	if !reflect.DeepEqual(meta, want) {
		t.Errorf("meta = %#v", meta)
	}
}

func TestFrontmatterEdges(t *testing.T) {
	meta, _, body, err := ParseFrontmatter("# T\n")
	if meta != nil || err != nil || body != "# T\n" {
		t.Errorf("no frontmatter: %#v %q %v", meta, body, err)
	}

	meta, raw, body, err := ParseFrontmatter("\ufeff---\r\ntype: term\r\n---\r\n# T\r\n")
	if err != nil || meta["type"] != "term" || raw != "type: term\r\n" || body != "# T\r\n" {
		t.Errorf("BOM and CRLF: %#v %q %q %v", meta, raw, body, err)
	}

	_, raw, body, err = ParseFrontmatter("---\ntype: [a\n---\nbody\n")
	if err == nil || raw != "type: [a\n" || body != "body\n" {
		t.Errorf("invalid YAML: %q %q %v", raw, body, err)
	}

	_, _, _, err = ParseFrontmatter("---\n- a\n- b\n---\n")
	if err == nil || !strings.Contains(err.Error(), "not a mapping") {
		t.Errorf("sequence root: %v", err)
	}

	_, _, _, err = ParseFrontmatter("---\nplain text\n---\n")
	if err == nil || !strings.Contains(err.Error(), "not a mapping") {
		t.Errorf("scalar root: %v", err)
	}

	meta, _, body, err = ParseFrontmatter("---\ntype: term\n")
	if meta != nil || err != nil || body != "---\ntype: term\n" {
		t.Errorf("never closed: %#v %q %v", meta, body, err)
	}
}

func TestFrontmatterEmptyIsNonNil(t *testing.T) {
	for _, text := range []string{"---\n---\n", "---\n# only a comment\n---\nx\n"} {
		meta, _, _, err := ParseFrontmatter(text)
		if err != nil || meta == nil || len(meta) != 0 {
			t.Errorf("%q: %#v %v", text, meta, err)
		}
	}
}

func TestFrontmatterClosingLineWithoutNewline(t *testing.T) {
	meta, raw, body, err := ParseFrontmatter("---\na: b\n---")
	if err != nil || meta["a"] != "b" || raw != "a: b\n" || body != "" {
		t.Errorf("%#v %q %q %v", meta, raw, body, err)
	}
}

func TestFrontmatterDelimiterMustBeExact(t *testing.T) {
	meta, _, _, err := ParseFrontmatter("--- \na: b\n---\n")
	if meta != nil || err != nil {
		t.Errorf("opening with trailing space: %#v %v", meta, err)
	}
	_, raw, body, _ := ParseFrontmatter("---\na: b\n----\n--- \n---\nrest\n")
	if raw != "a: b\n----\n--- \n" || body != "rest\n" {
		t.Errorf("only an exact line closes: %q %q", raw, body)
	}
}

func TestFrontmatterBodyStripsBOMWithoutFrontmatter(t *testing.T) {
	_, _, body, _ := ParseFrontmatter("\ufeff\ufeff# T\n")
	if body != "# T\n" {
		t.Errorf("body = %q", body)
	}
}

func TestFrontmatterValues(t *testing.T) {
	meta, _, _, err := ParseFrontmatter("---\nn: ~\nm: null\nnum: 12\nq: \"2026-01-01\"\nlist: [null, x, 3]\nnested: [a, [b]]\nmap: {k: v}\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"n":      "",
		"m":      "",
		"num":    "12",
		"q":      "2026-01-01",
		"list":   []string{"", "x", "3"},
		"nested": []any{"a", []any{"b"}},
		"map":    map[string]any{"k": "v"},
	}
	if !reflect.DeepEqual(meta, want) {
		t.Errorf("meta = %#v", meta)
	}
}

func TestFrontmatterDuplicateKey(t *testing.T) {
	_, _, _, err := ParseFrontmatter("---\na: 1\na: 2\n---\n")
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("err = %v", err)
	}
}
