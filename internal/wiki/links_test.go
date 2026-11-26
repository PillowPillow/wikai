package wiki

import (
	"reflect"
	"strings"
	"testing"
)

func TestMarkdownLinks(t *testing.T) {
	text := "See [a](../x/a%20b.md#Sec) and ![i](<img one.png> \"t\") and [u](https://example.com) and [s](#top).\n" +
		"`[c](code.md)` text\n```\n[f](fenced.md)\n```\nbroken [x](a b c\n"
	want := []Link{
		{Line: 1, Target: "../x/a b.md", Anchor: "Sec", HasAnchor: true},
		{Line: 1, Target: "img one.png"},
		{Line: 1, Target: "https://example.com", External: true},
		{Line: 1, Target: "", Anchor: "top", HasAnchor: true},
		{Line: 6, Unparsed: true},
	}
	if got := Links("markdown", text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestMarkdownLinkTargets(t *testing.T) {
	text := "[q](p.md?x=1#a%C3%A9) [z](a%zz+b.md) [e]() [m](mailto:x@example.com) [t](x.md 'title')\n"
	want := []Link{
		{Line: 1, Target: "p.md", Anchor: "aé", HasAnchor: true},
		{Line: 1, Target: "a%zz+b.md"},
		{Line: 1, Target: "", External: true},
		{Line: 1, Target: "mailto:x@example.com", External: true},
		{Line: 1, Target: "x.md"},
	}
	if got := Links("markdown", text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestWikilinks(t *testing.T) {
	text := "---\nsources: [\"[[Raw note]]\"]\n---\n[[Page]] [[Dir/Page#^b1|shown]] ![[img.png]] `[[code]]` | [[T\\|x]] |\n"
	want := []Link{
		{Line: 2, Target: "Raw note"},
		{Line: 4, Target: "Page"},
		{Line: 4, Target: "Dir/Page", Anchor: "^b1", HasAnchor: true},
		{Line: 4, Target: "img.png"},
		{Line: 4, Target: "T"},
	}
	if got := Links("wikilink", text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestWikilinksCodeOnlyInBody(t *testing.T) {
	text := "---\nx: \"`[[A]]`\"\nbad: [\n---\n```\n---\n[[Fenced]]\n```\n[[ B ]] | [[C#^b\\|x]] |\n"
	want := []Link{
		{Line: 2, Target: "A"},
		{Line: 9, Target: "B"},
		{Line: 9, Target: "C", Anchor: "^b", HasAnchor: true},
	}
	if got := Links("wikilink", text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestRewriteLinks(t *testing.T) {
	md := RewriteLinks("markdown", "[a](old.md#s) `[a](old.md)`", func(l Link) (string, bool) {
		if l.Target == "old.md" {
			return "new/n.md", true
		}
		return "", false
	})
	if md != "[a](new/n.md#s) `[a](old.md)`" {
		t.Errorf("markdown: %q", md)
	}
	wl := RewriteLinks("wikilink", "[[Old#^b|t]] [[Other]]", func(l Link) (string, bool) {
		if l.Target == "Old" {
			return "New", true
		}
		return "", false
	})
	if wl != "[[New#^b|t]] [[Other]]" {
		t.Errorf("wikilink: %q", wl)
	}
}

func TestRewriteMarkdownKeepsQueryTitleAndBrackets(t *testing.T) {
	text := "\ufeffA [a](old.md?v=1#s \"t\")\r\n```\n[f](old.md)\n```\n[b](<old.md>) [c](https://x/old.md) [d](old.md\r\n"
	got := RewriteLinks("markdown", text, func(l Link) (string, bool) {
		if l.Target == "old.md" {
			return "n w.md", true
		}
		return "", false
	})
	want := "\ufeffA [a](n%20w.md?v=1#s \"t\")\r\n```\n[f](old.md)\n```\n[b](<n w.md>) [c](https://x/old.md) [d](old.md\r\n"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestRewriteMarkdownRoundTrip(t *testing.T) {
	for _, odd := range []string{"dir (1)/50% #a?b<c>.md", "Re: x.md"} {
		for _, text := range []string{"[a](old.md#s \"t\")\n", "[a](<old.md>)\n"} {
			got := RewriteLinks("markdown", text, func(Link) (string, bool) { return odd, true })
			links := Links("markdown", got)
			if len(links) != 1 || links[0].Target != odd || links[0].Unparsed || links[0].External {
				t.Errorf("%q → %q → %+v", text, got, links)
			}
		}
	}
}

func TestRewriteWikilinks(t *testing.T) {
	text := "---\r\nsources: [\"[[Old]]\"]\r\n---\r\n![[Old]] | [[ Old \\|x]] |\r\n`[[Old]]`\r\n```\r\n[[Old]]\r\n```\r\n[[Old#H]]"
	got := RewriteLinks("wikilink", text, func(l Link) (string, bool) {
		if l.Target == "Old" {
			return "Dir/New", true
		}
		return "", false
	})
	want := "---\r\nsources: [\"[[Dir/New]]\"]\r\n---\r\n![[Dir/New]] | [[ Dir/New \\|x]] |\r\n`[[Old]]`\r\n```\r\n[[Old]]\r\n```\r\n[[Dir/New#H]]"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestRewriteLinksSeesWhatLinksReturns(t *testing.T) {
	cases := map[string]string{
		"markdown": "---\nx: 1\n---\n[a](a.md) `[b](b.md)` [c](https://c)\n```\n[d](d.md)\n```\n[e](#e) [f](f g\n",
		"wikilink": "---\ns: \"[[A]]\"\n---\n[[B]] `[[C]]`\n~~~\n[[D]]\n~~~\n[[E#^e|t]]\n",
	}
	for style, text := range cases {
		var seen []Link
		out := RewriteLinks(style, text, func(l Link) (string, bool) {
			seen = append(seen, l)
			return "", false
		})
		var parsed []Link
		for _, l := range Links(style, text) {
			if !l.Unparsed {
				parsed = append(parsed, l)
			}
		}
		if !reflect.DeepEqual(seen, parsed) {
			t.Errorf("%s: seen %+v, Links %+v", style, seen, parsed)
		}
		if out != text {
			t.Errorf("%s: text changed: %q", style, out)
		}
	}
}

func TestMarkdownLinkTextWithEscapedBrackets(t *testing.T) {
	want := []Link{{Line: 1, Target: "glossary/foo.md"}, {Line: 1, Target: "b.md"}}
	if got := Links("markdown", `- [Foo \[beta\]](glossary/foo.md) [a\\](b.md)`); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestLinksUnknownStyle(t *testing.T) {
	if got := Links("other", "[a](b) [[c]]"); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
	if got := RewriteLinks("other", "[a](b)", func(Link) (string, bool) { return "x", true }); !strings.Contains(got, "(b)") {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownLinkAfterEscapedBacktick(t *testing.T) {
	want := []Link{{Line: 1, Target: "glossary/foo.md"}}
	if got := Links("markdown", "- [a\\`b](glossary/foo.md) — c`d"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}
