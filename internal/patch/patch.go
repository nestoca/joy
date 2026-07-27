// Package patch applies a subset of RFC 6902 JSON Patch (add / replace / remove) directly
// against a gopkg.in/yaml.v3 node tree. Operating on nodes (rather than a JSON round-trip)
// preserves the catalog's custom YAML tags (e.g. !lock) and node styles.
package patch

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Op is a single RFC 6902 operation.
type Op struct {
	Op    string    `yaml:"op" json:"op"`
	Path  string    `yaml:"path" json:"path"`
	Value yaml.Node `yaml:"value" json:"value"`
}

// ParseOp decodes a single RFC 6902 op (YAML or JSON object).
func ParseOp(data []byte) (Op, error) {
	var op Op
	if strings.TrimSpace(string(data)) == "" {
		return op, fmt.Errorf("empty patch op")
	}
	if err := yaml.Unmarshal(data, &op); err != nil {
		return op, fmt.Errorf("decoding patch op: %w", err)
	}
	// Normalize style so JSON-authored values (e.g. flow maps, double-quoted scalars) render as
	// clean block YAML in the catalog. The encoder still re-quotes any scalar that needs it.
	clearStyle(&op.Value)
	return op, nil
}

func clearStyle(node *yaml.Node) {
	if node == nil {
		return
	}
	node.Style = 0
	for _, child := range node.Content {
		clearStyle(child)
	}
}

// Apply runs each op against root (the mapping node, e.g. the document's first content node).
func Apply(root *yaml.Node, ops []Op) error {
	for i, op := range ops {
		if err := applyOp(root, op); err != nil {
			return fmt.Errorf("patch op %d (%s %s): %w", i, op.Op, op.Path, err)
		}
	}
	return nil
}

func applyOp(root *yaml.Node, op Op) error {
	tokens, err := parsePointer(op.Path)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return fmt.Errorf("cannot target the document root")
	}
	parent, last, err := resolveParent(root, tokens)
	if err != nil {
		if op.Op == "add" {
			return fmt.Errorf("%w (RFC 6902 'add' requires the parent to exist; add the parent object instead)", err)
		}
		return err
	}
	switch op.Op {
	case "add":
		return addChild(parent, last, &op.Value)
	case "replace":
		return replaceChild(parent, last, &op.Value)
	case "remove":
		return removeChild(parent, last, false)
	case "move", "copy", "test":
		return fmt.Errorf("op %q is not supported (only add/replace/remove)", op.Op)
	default:
		return fmt.Errorf("unknown op %q", op.Op)
	}
}

// SetPath sets tokens (a decomposed path) to val, adding or replacing as needed.
func SetPath(root *yaml.Node, tokens []string, val *yaml.Node) error {
	parent, last, err := resolveParent(root, tokens)
	if err != nil {
		return err
	}
	if parent.Kind != yaml.MappingNode {
		return fmt.Errorf("SetPath %v: parent is not a mapping", tokens)
	}
	return setMapping(parent, last, val, false)
}

// SetPathCreating sets tokens to val, creating any missing intermediate mapping nodes along
// the way (merging into existing ones rather than clobbering). Used for built-in transforms
// that may target not-yet-existing parents (e.g. metadata.labels).
func SetPathCreating(root *yaml.Node, tokens []string, val *yaml.Node) error {
	cur := root
	for _, t := range tokens[:len(tokens)-1] {
		if cur.Kind != yaml.MappingNode {
			return fmt.Errorf("SetPathCreating %v: %q is not a mapping", tokens, t)
		}
		if i, ok := mappingIndex(cur, t); ok {
			cur = cur.Content[i+1]
			continue
		}
		child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		cur.Content = append(cur.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t}, child)
		cur = child
	}
	if cur.Kind != yaml.MappingNode {
		return fmt.Errorf("SetPathCreating %v: parent is not a mapping", tokens)
	}
	return setMapping(cur, tokens[len(tokens)-1], val, false)
}

// Scalar builds a plain string scalar node.
func Scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func addChild(parent *yaml.Node, key string, val *yaml.Node) error {
	if val == nil || val.Kind == 0 {
		return fmt.Errorf("add: missing value")
	}
	switch parent.Kind {
	case yaml.MappingNode:
		return setMapping(parent, key, val, false)
	case yaml.SequenceNode:
		if key == "-" {
			parent.Content = append(parent.Content, val)
			return nil
		}
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx > len(parent.Content) {
			return fmt.Errorf("add: invalid array index %q", key)
		}
		parent.Content = slices.Insert(parent.Content, idx, val)
		return nil
	default:
		return fmt.Errorf("add: parent is not a container")
	}
}

func replaceChild(parent *yaml.Node, key string, val *yaml.Node) error {
	if val == nil || val.Kind == 0 {
		return fmt.Errorf("replace: missing value")
	}
	switch parent.Kind {
	case yaml.MappingNode:
		return setMapping(parent, key, val, true)
	case yaml.SequenceNode:
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(parent.Content) {
			return fmt.Errorf("replace: array index %q out of range", key)
		}
		parent.Content[idx] = preserveScalar(parent.Content[idx], val)
		return nil
	default:
		return fmt.Errorf("replace: parent is not a container")
	}
}

func removeChild(parent *yaml.Node, key string, tolerant bool) error {
	switch parent.Kind {
	case yaml.MappingNode:
		if i, ok := mappingIndex(parent, key); ok {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return nil
		}
		if tolerant {
			return nil
		}
		return fmt.Errorf("remove: key %q not found", key)
	case yaml.SequenceNode:
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(parent.Content) {
			if tolerant {
				return nil
			}
			return fmt.Errorf("remove: array index %q out of range", key)
		}
		parent.Content = slices.Delete(parent.Content, idx, idx+1)
		return nil
	default:
		return fmt.Errorf("remove: parent is not a container")
	}
}

// setMapping adds or replaces key in a mapping node. When the key exists and both the old and
// new values are scalars, the old node's tag/style/comments are preserved (so replacing a
// !lock value keeps !lock).
func setMapping(m *yaml.Node, key string, val *yaml.Node, mustExist bool) error {
	if i, ok := mappingIndex(m, key); ok {
		m.Content[i+1] = preserveScalar(m.Content[i+1], val)
		return nil
	}
	if mustExist {
		return fmt.Errorf("key %q not found", key)
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
	return nil
}

func preserveScalar(old, val *yaml.Node) *yaml.Node {
	if old != nil && old.Kind == yaml.ScalarNode && val.Kind == yaml.ScalarNode {
		c := *val
		c.Tag = old.Tag
		c.Style = old.Style
		c.HeadComment, c.LineComment, c.FootComment = old.HeadComment, old.LineComment, old.FootComment
		return &c
	}
	return val
}

func mappingIndex(m *yaml.Node, key string) (int, bool) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i, true
		}
	}
	return -1, false
}

// resolveParent walks all but the last token and returns the container + last token.
func resolveParent(root *yaml.Node, tokens []string) (*yaml.Node, string, error) {
	cur := root
	for _, t := range tokens[:len(tokens)-1] {
		next, err := child(cur, t)
		if err != nil {
			return nil, "", err
		}
		cur = next
	}
	return cur, tokens[len(tokens)-1], nil
}

func child(n *yaml.Node, token string) (*yaml.Node, error) {
	switch n.Kind {
	case yaml.MappingNode:
		if i, ok := mappingIndex(n, token); ok {
			return n.Content[i+1], nil
		}
		return nil, fmt.Errorf("key %q not found", token)
	case yaml.SequenceNode:
		idx, err := strconv.Atoi(token)
		if err != nil || idx < 0 || idx >= len(n.Content) {
			return nil, fmt.Errorf("array index %q out of range", token)
		}
		return n.Content[idx], nil
	default:
		return nil, fmt.Errorf("cannot descend into %q", token)
	}
}

// parsePointer splits an RFC 6901 JSON Pointer into unescaped tokens.
func parsePointer(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid JSON pointer %q: must start with '/'", path)
	}
	parts := strings.Split(path, "/")[1:]
	for i, p := range parts {
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts, nil
}
