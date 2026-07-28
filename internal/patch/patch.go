// Package patch applies RFC 6902 JSON Patch operations to a YAML release tree.
//
// The patch itself is applied by a standard JSON Patch library, so serialization to JSON is
// required — which drops YAML custom tags (e.g. !lock), comments and key order. Those are
// restored from the original tree afterwards via yml.CopyMetadata. It also exposes a few
// node-level setters used for built-in transforms that must run before the JSON round-trip.
package patch

import (
	"bytes"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/nestoca/joy/internal/yml"
)

// Op is a single RFC 6902 operation, stored as its JSON encoding.
type Op struct {
	json []byte
}

// ParseOp decodes a single RFC 6902 op (YAML or JSON object) into its JSON form.
func ParseOp(data []byte) (Op, error) {
	j, err := sigsyaml.YAMLToJSON(data)
	if err != nil {
		return Op{}, fmt.Errorf("decoding patch op: %w", err)
	}
	if trimmed := bytes.TrimSpace(j); len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return Op{}, fmt.Errorf("empty patch op")
	}
	return Op{json: j}, nil
}

// Apply applies the ops to doc (a document node) and returns the resulting document. The patch is
// applied via a JSON Patch library; custom tags, comments and key order are then restored from
// doc, and styles are cleared so the result serializes as clean block YAML. When there are no
// ops, doc is returned unchanged.
func Apply(doc *yaml.Node, ops []Op) (*yaml.Node, error) {
	if len(ops) == 0 {
		return doc, nil
	}

	docJSON, err := toJSON(doc)
	if err != nil {
		return nil, fmt.Errorf("converting release to json: %w", err)
	}

	patch, err := jsonpatch.DecodePatch(patchArray(ops))
	if err != nil {
		return nil, fmt.Errorf("decoding patch: %w", err)
	}

	patchedJSON, err := patch.Apply(docJSON)
	if err != nil {
		return nil, fmt.Errorf("applying patch: %w", err)
	}

	var patched yaml.Node
	if err := yaml.Unmarshal(patchedJSON, &patched); err != nil {
		return nil, fmt.Errorf("parsing patched result: %w", err)
	}

	// The JSON round-trip drops custom tags, comments and key order, and flow-flattens styles.
	yml.CopyMetadata(&patched, doc)
	clearStyle(&patched)
	return &patched, nil
}

// patchArray concatenates the ops' JSON objects into a single JSON Patch array.
func patchArray(ops []Op) []byte {
	parts := make([][]byte, len(ops))
	for i, op := range ops {
		parts[i] = op.json
	}
	return append(append([]byte{'['}, bytes.Join(parts, []byte{','})...), ']')
}

func toJSON(node *yaml.Node) ([]byte, error) {
	data, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}
	return sigsyaml.YAMLToJSON(data)
}

// Scalar builds a plain string scalar node.
func Scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// SetPath sets a mapping value at the given path (tokens), adding or replacing as needed. Every
// intermediate node must already exist and be a mapping. Used for built-in transforms.
func SetPath(root *yaml.Node, tokens []string, val *yaml.Node) error {
	parent, last, err := resolveParent(root, tokens)
	if err != nil {
		return err
	}
	return setMapping(parent, last, val)
}

// SetPathCreating is like SetPath but creates any missing intermediate mapping nodes (merging
// into existing ones rather than clobbering). Used for built-ins that may target absent parents
// (e.g. metadata.labels).
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
	return setMapping(cur, tokens[len(tokens)-1], val)
}

func resolveParent(root *yaml.Node, tokens []string) (*yaml.Node, string, error) {
	if len(tokens) == 0 {
		return nil, "", fmt.Errorf("empty path")
	}
	cur := root
	for _, t := range tokens[:len(tokens)-1] {
		if cur.Kind != yaml.MappingNode {
			return nil, "", fmt.Errorf("path segment %q: parent is not a mapping", t)
		}
		i, ok := mappingIndex(cur, t)
		if !ok {
			return nil, "", fmt.Errorf("path segment %q not found", t)
		}
		cur = cur.Content[i+1]
	}
	return cur, tokens[len(tokens)-1], nil
}

// setMapping adds or replaces key in a mapping node. When both old and new values are scalars,
// the old node's tag/style/comments are preserved (so replacing a !lock value keeps !lock).
func setMapping(m *yaml.Node, key string, val *yaml.Node) error {
	if m.Kind != yaml.MappingNode {
		return fmt.Errorf("cannot set key %q: parent is not a mapping", key)
	}
	if i, ok := mappingIndex(m, key); ok {
		m.Content[i+1] = preserveScalar(m.Content[i+1], val)
		return nil
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

func clearStyle(node *yaml.Node) {
	if node == nil {
		return
	}
	node.Style = 0
	for _, child := range node.Content {
		clearStyle(child)
	}
}
