package promote

import (
	"slices"

	"gopkg.in/yaml.v3"
)

func Merge(dst, src *yaml.Node) *yaml.Node {
	dst = clone(dst)

	doc := func() *yaml.Node {
		if dst.Kind == yaml.DocumentNode {
			return dst
		}
		return &yaml.Node{Kind: yaml.DocumentNode}
	}()

	dst = unwrapDocument(dst)
	src = markLockedValuesAsTodo(clone(unwrapDocument(src)), false)

	doc.Content = []*yaml.Node{merge(dst, src)}
	return doc
}

func merge(dst, src *yaml.Node) *yaml.Node {
	// If destination is locked, it does not matter what source is.
	// If destination exists but source is locked, disregard source.
	if isLocked(dst) || (dst != nil && isLocked(src)) {
		return dst
	}

	// If destination is nil, create the node from source.
	// If source is nil we can set the return to nil which will remove the node
	// in map and sequence merges. This is fine because we know dst is not locked.
	// If the kind is different we simply go with the updating source.
	if dst == nil || src == nil || dst.Kind != src.Kind {
		return src
	}

	switch src.Kind {
	case yaml.MappingNode:
		return mergeMap(dst, src)
	case yaml.SequenceNode:
		return mergeSeq(dst, src)
	default: // Scalar and Alias nodes
		return src
	}
}

func mergeMap(dst, src *yaml.Node) *yaml.Node {
	var (
		dstMap  = asMap(dst)
		srcMap  = asMap(src)
		keys    = dedup(append(keysOf(src), keysOf(dst)...))
		content = make([]*yaml.Node, 0, len(src.Content)+len(dst.Content))
	)

	for _, key := range keys {
		if value := merge(dstMap[key], srcMap[key]); value != nil {
			content = append(content, strNode(key), value)
		}
	}

	src.Content = content
	return src
}

func mergeSeq(dst, src *yaml.Node) *yaml.Node {
	maxLen := max(len(dst.Content), len(src.Content))
	content := make([]*yaml.Node, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		if value := merge(at(dst, i), at(src, i)); value != nil {
			content = append(content, value)
		}
	}
	src.Content = content
	return src
}

func unwrapDocument(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		node = node.Content[0]
	}
	return node
}

func at(node *yaml.Node, i int) *yaml.Node {
	if i >= len(node.Content) {
		return nil
	}
	return node.Content[i]
}

func clone(node *yaml.Node) *yaml.Node {
	copy := *node
	copy.Content = make([]*yaml.Node, len(node.Content))

	for i, node := range node.Content {
		copy.Content[i] = clone(node)
	}
	return &copy
}

func markLockedValuesAsTodo(node *yaml.Node, locked bool) *yaml.Node {
	locked = locked || isLocked(node)

	switch node.Kind {
	case yaml.ScalarNode:
		if locked {
			node.Value = "TODO"
		}
	case yaml.SequenceNode, yaml.MappingNode:
		for i, n := range node.Content {
			node.Content[i] = markLockedValuesAsTodo(n, locked)
		}
	}

	return node
}

func isLocked(node *yaml.Node) bool {
	return node != nil && node.Tag == "!lock"
}

func asMap(node *yaml.Node) map[string]*yaml.Node {
	result := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		result[node.Content[i].Value] = node.Content[i+1]
	}
	return result
}

func keysOf(node *yaml.Node) []string {
	keys := make([]string, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	return keys
}

func dedup(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func strNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}
