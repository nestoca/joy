package yml

import (
	"gopkg.in/yaml.v3"
)

// Merge merges the source release into the destination release, preserving the
// destination release's locked values. Source and destination releases are
// left unchanged and a new release node tree is returned.
func Merge(src *yaml.Node, dest *yaml.Node) *yaml.Node {
	if src == nil {
		src = &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{{
				Kind: yaml.MappingNode,
			}},
		}
	}

	result := DeepCopyNode(src)

	if dest == nil {
		dest = &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{{
				Kind: yaml.MappingNode,
			}},
		}
	} else {
		dest = DeepCopyNode(dest)
	}

	if src.Kind != yaml.DocumentNode || dest.Kind != yaml.DocumentNode {
		return nil
	}

	setLockedScalarValuesAsTodo(result.Content[0], false)

	merged := mergeSubTrees(result.Content[0], dest.Content[0])
	if merged != nil {
		result.Content[0] = merged
	}
	return result
}

// mergeSubTrees merges the locked subtrees from dest into src, which is basically equivalent to
// merging src into dest, but preserving locked values in dest.
func mergeSubTrees(src, dest *yaml.Node) *yaml.Node {
	lockMarkerFound := false

	if src == nil {
		src = &yaml.Node{
			Kind:        yaml.MappingNode,
			Content:     []*yaml.Node{},
			HeadComment: dest.HeadComment,
			LineComment: dest.LineComment,
			FootComment: dest.FootComment,
		}
	}

	if src.Kind != yaml.MappingNode || dest.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(dest.Content)-1; i += 2 {
		// Read key and value
		destKeyNode := dest.Content[i]
		destValueNode := dest.Content[i+1]
		if destKeyNode.Kind != yaml.ScalarNode {
			continue
		}
		key := destKeyNode.Value

		// Find source location
		srcIndex := findKey(src, key)
		var srcValueNode *yaml.Node
		var srcKeyNode *yaml.Node
		if srcIndex != -1 {
			srcKeyNode = src.Content[srcIndex]
			srcValueNode = src.Content[srcIndex+1]
		}

		var subtree *yaml.Node
		if IsLocked(destKeyNode, destValueNode) || (srcValueNode != nil && IsLocked(srcKeyNode, srcValueNode)) {
			lockMarkerFound = true
			subtree = destValueNode
		} else {
			subtree = mergeSubTrees(srcValueNode, destValueNode)
		}

		// Copy source lock comment if any
		if destKeyNode != nil && IsLocked(srcKeyNode, srcValueNode) {
			destKeyNode.HeadComment = srcKeyNode.HeadComment
			destKeyNode.LineComment = srcKeyNode.LineComment
		}

		// Was a subtree with some locked nodes in it found?
		if subtree != nil {
			lockMarkerFound = true
			// Are we overwriting an existing node?
			if srcIndex != -1 {
				src.Content[srcIndex] = destKeyNode
				src.Content[srcIndex+1] = subtree
			} else {
				// We are adding a new node
				insertIndex := 0
				if i > 0 {
					insertIndex = findInsertionIndex(src, dest, key)
				}
				if insertIndex != -1 {
					// Insert at the correct position
					src.Content = append(src.Content[:insertIndex], append([]*yaml.Node{destKeyNode, subtree}, src.Content[insertIndex:]...)...)
				} else {
					// Append at the end
					src.Content = append(src.Content, destKeyNode, subtree)
				}
			}
		}
	}

	if lockMarkerFound {
		src.Style = dest.Style
		return src
	}
	return nil
}

func findInsertionIndex(src, dest *yaml.Node, key string) int {
	// Look at all keys in dest that come before the key we are looking for
	// and find the first index in src that comes after them.
	index := -1
	for i := 0; i < len(dest.Content)-1; i += 2 {
		// Read dest key
		destKeyNode := dest.Content[i]
		if destKeyNode.Kind != yaml.ScalarNode {
			continue
		}
		destKey := destKeyNode.Value

		// We've reached the key we are looking for, we're done
		if destKey == key {
			break
		}

		// Look in src for the key
		srcIndex := findKey(src, destKey)
		if srcIndex != -1 {
			// We should insert _at_least_ after that key
			index = srcIndex + 2
		}
	}
	return index
}

// setLockedScalarValuesAsTodo sets all scalar values in locked subtrees to "TODO" to remind developers to manually
// update them to environment-specific values.
func setLockedScalarValuesAsTodo(node *yaml.Node, locked bool) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if locked {
				if valueNode.Kind == yaml.ScalarNode {
					valueNode.Value = "TODO"
					valueNode.Style = 0
				} else {
					setLockedScalarValuesAsTodo(valueNode, true)
				}
			} else {
				setLockedScalarValuesAsTodo(valueNode, IsLocked(keyNode, valueNode))
			}
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			setLockedScalarValuesAsTodo(item, locked)
		}
	}
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
