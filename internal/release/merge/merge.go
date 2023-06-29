package merge

import (
	"gopkg.in/yaml.v3"
	"regexp"
)

var lockMarkerRegex = regexp.MustCompile(`(?m)^##\s*lock\s*$`)

// Merge merges the locked subtrees from src onto dest.
func Merge(dest *yaml.Node, src *yaml.Node) {
	if dest.Kind != yaml.DocumentNode || src.Kind != yaml.DocumentNode {
		return
	}

	result := mergeSubTrees(dest.Content[0], src.Content[0])
	if result != nil {
		dest.Content[0] = result
	}
}

func mergeSubTrees(dest, src *yaml.Node) *yaml.Node {
	lockMarkerFound := false

	if dest == nil {
		dest = &yaml.Node{
			Kind:        yaml.MappingNode,
			Content:     []*yaml.Node{},
			HeadComment: src.HeadComment,
			LineComment: src.LineComment,
			FootComment: src.FootComment,
		}
	}

	if dest.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(src.Content)-1; i += 2 {
		// Read key and value
		keyNode := src.Content[i]
		valueNode := src.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode {
			continue
		}
		key := keyNode.Value

		// Find destination location
		destIndex := findKey(dest, key)
		var destValueNode *yaml.Node
		if destIndex != -1 {
			destValueNode = dest.Content[destIndex+1]
		}

		var subtree *yaml.Node
		isKeyNodeMarkedAsLocked :=
			lockMarkerRegex.MatchString(keyNode.HeadComment) ||
				lockMarkerRegex.MatchString(keyNode.LineComment)
		isValueNodeMarkedAsLocked :=
			valueNode.Kind == yaml.ScalarNode && lockMarkerRegex.MatchString(valueNode.LineComment)
		if isKeyNodeMarkedAsLocked || isValueNodeMarkedAsLocked {
			lockMarkerFound = true
			subtree = valueNode
		} else {
			subtree = mergeSubTrees(destValueNode, valueNode)
		}

		// Was a subtree with some locked nodes in it found?
		if subtree != nil {
			lockMarkerFound = true
			// Are we overwriting an existing node?
			if destIndex != -1 {
				dest.Content[destIndex] = keyNode
				dest.Content[destIndex+1] = subtree
			} else {
				// We are adding a new node
				dest.Content = append(dest.Content, keyNode, subtree)
			}
		}
	}

	if lockMarkerFound {
		dest.Style = src.Style
		return dest
	}
	return nil
}

func findKey(node *yaml.Node, key string) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == key {
			return i
		}
	}
	return -1
}
