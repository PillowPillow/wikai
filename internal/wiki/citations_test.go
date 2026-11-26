package wiki

import (
	"reflect"
	"testing"
)

func TestCitations(t *testing.T) {
	text := "A (source: repo@1a2b3c4:a.md#L1-L2). B (source: [doc](../x.md)). C (source: f(x) note). D (source: \n`(source: code)`\n"
	want := []Citation{
		{Line: 1, Key: "repo@1a2b3c4:a.md#L1-L2"},
		{Line: 1, Key: "../x.md"},
		{Line: 1, Key: "f(x) note"},
		{Line: 1, Unparsed: true},
	}
	if got := Citations(text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestCitationEdges(t *testing.T) {
	text := "(source  :  x  )\n" +
		"(source: [d]( ../y.md ) )\n" +
		"(source: [d](x) extra)\n" +
		"(source: f(g(x)))\n" +
		"(source: )\n" +
		"```\n(source: fenced)\n```\n" +
		"(Source: x) (source x)\n"
	want := []Citation{
		{Line: 1, Key: "x"},
		{Line: 2, Key: "../y.md"},
		{Line: 3, Unparsed: true},
		{Line: 4, Unparsed: true},
		{Line: 5, Key: ""},
	}
	if got := Citations(text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}
