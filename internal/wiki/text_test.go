package wiki

import (
	"reflect"
	"strings"
	"testing"
)

func TestAnchors(t *testing.T) {
	got := Anchors("# Définition\n## Two words\n## Two words\n```\n# not a heading\n```\n")
	want := map[string]bool{"définition": true, "two-words": true, "two-words-1": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestAnchorsKeepCodeSpanText(t *testing.T) {
	got := Anchors("## The `x` flag: on/off_2 ##\n")
	if !got["the-x-flag-onoff_2"] || len(got) != 1 {
		t.Errorf("got %v", got)
	}
}

func TestHeadings(t *testing.T) {
	got := Headings("# A #\n    # indented four\n   ### B\n#nospace\n~~~~\n# in\n~~~\n# still in\n~~~~~\n## C\n")
	want := []string{"A", "B", "C"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestHeadingTitle(t *testing.T) {
	title, ok := HeadingTitle("intro\n```\n# fenced\n```\n## Sub\n# The `x` title ##\n# Second\n")
	if !ok || title != "The `x` title" {
		t.Errorf("got %q %v", title, ok)
	}
	if _, ok := HeadingTitle("## only sub\n"); ok {
		t.Error("no title expected")
	}
}

func TestOutsideCode(t *testing.T) {
	got := OutsideCode("a `b` c\r\n```go\nx\n````\ny\n```\nz\n```\nd ``e ` f`` g `h\n")
	want := []Line{{1, "a     c"}, {5, "y"}, {9, "d" + strings.Repeat(" ", 11) + "g `h"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestOutsideCodeFenceNeedsSameCharAndLength(t *testing.T) {
	got := OutsideCode("````\n```\n~~~~\n````\nout\n")
	want := []Line{{5, "out"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestOutsideCodeUnmatchedRunIsLiteral(t *testing.T) {
	got := OutsideCode("``a` b `c`\n")
	want := []Line{{1, "``a     c`"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestOutsideCodeBlanksMultibyteByByte(t *testing.T) {
	got := OutsideCode("`é` x\n")
	if len(got) != 1 || got[0].Text != "     x" {
		t.Errorf("got %+v", got)
	}
}

func TestHasBlockID(t *testing.T) {
	if !HasBlockID("first\nline ^abc-1  \nlast", "abc-1") {
		t.Error("block id at end of line not found")
	}
	if HasBlockID("^abc-1 mid line\n", "abc-1") {
		t.Error("block id mid line found")
	}
	if !HasBlockID("x ^b1\r\n", "b1") {
		t.Error("block id before CRLF not found")
	}
	if HasBlockID("x ^ab1\n", "b1") {
		t.Error("suffix of a longer id found")
	}
}
