package yml

import (
	"slices"

	"gopkg.in/yaml.v3"
)

func HasLockedTodos(node *yaml.Node) bool {
	return hasLockedTodos(node, false)
}

func hasLockedTodos(node *yaml.Node, locked bool) bool {
	locked = locked || IsLocked(node)

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			var (
				key   = node.Content[i]
				value = node.Content[i+1]
			)
			if hasLockedTodos(value, locked || IsLocked(key)) {
				return true
			}
		}
		return false
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, n := range node.Content {
			if hasLockedTodos(n, locked) {
				return true
			}
		}
		return false

	default:
		return locked && node.Value == "TODO"
	}
}

func GetMappingValueNodesWithTags(node *yaml.Node) (nodes []*yaml.Node) {
	if node.Kind == yaml.MappingNode {
		for i := 1; i < len(node.Content); i += 2 {
			value := node.Content[i]
			if slices.Contains(CustomTags, value.Tag) {
				nodes = append(nodes, value)
			}
		}
	}
	for _, node := range node.Content {
		nodes = append(nodes, GetMappingValueNodesWithTags(node)...)
	}
	return
}
