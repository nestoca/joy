package releasing

import (
	"gopkg.in/yaml.v3"
)

// DeepCopyNode returns a deep copy of the given node.
func DeepCopyNode(node *yaml.Node) *yaml.Node {
	clone := *node

	if node.Content != nil {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			clone.Content[i] = DeepCopyNode(child)
		}
	}

	return &clone
}
