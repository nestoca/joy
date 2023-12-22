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

func EqualWithoutLocks(a, b *yaml.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if isLocked(a) || isLocked(b) {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Kind != b.Kind || a.Value != b.Value {
		return false
	}

	if len(a.Content) != len(b.Content) {
		return false
	}

	for i := range a.Content {
		if !EqualWithoutLocks(a.Content[i], b.Content[i]) {
			return false
		}
	}

	return true
}
