package wiki

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

const bom = "\ufeff"

type frontmatterBounds struct {
	rawStart, rawEnd, bodyStart int
	closingLine                 int
}

func findFrontmatter(text string) (frontmatterBounds, bool) {
	lines := splitLines(text)
	if len(lines) == 0 || strings.TrimLeft(lines[0].text, bom) != "---" {
		return frontmatterBounds{}, false
	}
	for i := 1; i < len(lines); i++ {
		if lines[i].text != "---" {
			continue
		}
		body := len(text)
		if i+1 < len(lines) {
			body = lines[i+1].start
		}
		return frontmatterBounds{lines[1].start, lines[i].start, body, lines[i].number}, true
	}
	return frontmatterBounds{}, false
}

func ParseFrontmatter(text string) (meta map[string]any, raw, body string, err error) {
	b, ok := findFrontmatter(text)
	if !ok {
		return nil, "", strings.TrimLeft(text, bom), nil
	}
	raw, body = text[b.rawStart:b.rawEnd], text[b.bodyStart:]
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, raw, body, err
	}
	meta = map[string]any{}
	if len(doc.Content) == 0 {
		return meta, raw, body, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, raw, body, errors.New("not a mapping")
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if _, dup := meta[key]; dup {
			return nil, raw, body, fmt.Errorf("duplicate key %q", key)
		}
		v, err := nodeValue(root.Content[i+1])
		if err != nil {
			return nil, raw, body, err
		}
		meta[key] = v
	}
	return meta, raw, body, nil
}

func resolveAlias(n *yaml.Node) *yaml.Node {
	for n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	return n
}

func scalarText(n *yaml.Node) string {
	if n.ShortTag() == "!!null" {
		return ""
	}
	return n.Value
}

func nodeValue(n *yaml.Node) (any, error) {
	n = resolveAlias(n)
	switch n.Kind {
	case yaml.ScalarNode:
		return scalarText(n), nil
	case yaml.SequenceNode:
		if items, ok := scalarItems(n); ok {
			return items, nil
		}
	}
	var v any
	err := n.Decode(&v)
	return v, err
}

func scalarItems(n *yaml.Node) ([]string, bool) {
	items := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		c = resolveAlias(c)
		if c.Kind != yaml.ScalarNode {
			return nil, false
		}
		items = append(items, scalarText(c))
	}
	return items, true
}
