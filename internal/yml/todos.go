package yml

import "gopkg.in/yaml.v3"

func HasLockedTodos(node *yaml.Node) bool {
	return hasLockedTodos(node, false)
}

func hasLockedTodos(node *yaml.Node, locked bool) bool {
	locked = locked || isLocked(node)

	switch node.Kind {
	case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode:
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
