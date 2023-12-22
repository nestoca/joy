package yml

import "gopkg.in/yaml.v3"

// Compare returns true if the two yaml documents are equivalent in terms of contents and comments, false otherwise,
// disregarding differences in formatting / whitespace.
func Compare(yaml1, yaml2 []byte) bool {
	var node1, node2 yaml.Node

	err1 := yaml.Unmarshal(yaml1, &node1)
	err2 := yaml.Unmarshal(yaml2, &node2)

	if err1 != nil || err2 != nil {
		return false
	}

	return deepEqualNode(&node1, &node2)
}

func deepEqualNode(node1, node2 *yaml.Node) bool {
	if node1.Kind != node2.Kind ||
		node1.Value != node2.Value ||
		node1.HeadComment != node2.HeadComment ||
		node1.LineComment != node2.LineComment ||
		node1.FootComment != node2.FootComment {
		return false
	}

	if len(node1.Content) != len(node2.Content) {
		return false
	}

	for i := range node1.Content {
		if !deepEqualNode(node1.Content[i], node2.Content[i]) {
			return false
		}
	}

	return true
}

func EqualWithExclusion(a, b *yaml.Node, excludedPath ...string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handling document nodes
	if a.Kind == yaml.DocumentNode && b.Kind == yaml.DocumentNode {
		if !EqualWithExclusion(a.Content[0], b.Content[0], excludedPath...) {
			return false
		}
		return true
	}

	if a.Kind != b.Kind || a.Value != b.Value {
		return false
	}
	if len(a.Content) != len(b.Content) {
		return false
	}

	// Determine current key and rest of path
	excludedKey := ""
	if len(excludedPath) == 1 {
		excludedKey = excludedPath[0]
	}
	var excludedSubPath []string
	if len(excludedPath) > 0 {
		excludedSubPath = excludedPath[1:]
	}

	for i := 0; i < len(a.Content); i++ {
		if a.Kind == yaml.MappingNode && i%2 == 0 && a.Content[i].Value == excludedKey {
			// Skip this key and its corresponding value node
			i++
			continue
		}

		if !EqualWithExclusion(a.Content[i], b.Content[i], excludedSubPath...) {
			return false
		}
	}

	return true
}
