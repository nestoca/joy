package yml

import (
	"slices"

	"gopkg.in/yaml.v3"
)

func Merge(dst, src *yaml.Node) *yaml.Node {
	dst, src = Clone(dst), Clone(src)

	doc := func() *yaml.Node {
		if dst != nil && dst.Kind == yaml.DocumentNode {
			return dst
		}
		if src != nil && src.Kind == yaml.DocumentNode {
			return src
		}
		return &yaml.Node{Kind: yaml.DocumentNode}
	}()

	dst = unwrapDocument(dst)
	src = markLockedValuesAsTodo(unwrapDocument(src), false)

	result := merge(dst, src)
	if result == nil {
		result = &yaml.Node{Kind: yaml.MappingNode}
	}

	doc.Content = []*yaml.Node{result}
	return doc
}

func merge(dst, src *yaml.Node) *yaml.Node {
	// If destination is locked, it does not matter what source is.
	// If destination exists but source is locked, disregard source.
	if isLocked(dst) || (dst != nil && isLocked(src)) || isLocal(dst) || isLocal(src) {
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
		content []*yaml.Node
	)

	for _, key := range keys {
		dstKV, srcKV := dstMap[key], srcMap[key]
		if value := merge(dstKV.Value, srcKV.Value); value != nil {
			content = append(content, firstNonNil(dstKV.Key, srcKV.Key), value)
		}
	}

	dst.Content = content
	dst.Style = mergeStyle(dst.Style, src.Style)

	return dst
}

func mergeSeq(dst, src *yaml.Node) *yaml.Node {
	var (
		srcIdx  int
		dstIdx  int
		content []*yaml.Node
	)
	for {
		// If we have moved past both the source and dst length we must end the loop unless
		// we want to go to infinity and beyond!
		if srcIdx >= len(src.Content) && dstIdx >= len(dst.Content) {
			break
		}

		// For the destination if the item is local we want to add it to the result, but move the dstIdx
		// forward until we find a non local element to merge with.
		for {
			item := at(dst, dstIdx)
			if !isLocal(item) {
				break
			}
			content = append(content, item)
			dstIdx++
		}

		// ignore local source elements and find the first non local idx.
		for isLocal(at(src, srcIdx)) {
			srcIdx++
		}

		if value := merge(at(dst, dstIdx), at(src, srcIdx)); value != nil {
			content = append(content, value)
		}

		srcIdx++
		dstIdx++
	}

	dst.Content = content
	dst.Style = mergeStyle(dst.Style, src.Style)

	return dst
}

func at(node *yaml.Node, i int) *yaml.Node {
	if node == nil || i >= len(node.Content) {
		return nil
	}
	return node.Content[i]
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

func Clone(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	copy := *node
	if len(node.Content) > 0 {
		copy.Content = make([]*yaml.Node, len(node.Content))
		for i, node := range node.Content {
			copy.Content[i] = Clone(node)
		}
	}

	return &copy
}

func markLockedValuesAsTodo(node *yaml.Node, locked bool) *yaml.Node {
	if node == nil {
		return nil
	}

	locked = locked || isLocked(node)

	switch node.Kind {
	case yaml.ScalarNode:
		if locked {
			node.Value = "TODO"
		}
	case yaml.SequenceNode:
		for i, n := range node.Content {
			node.Content[i] = markLockedValuesAsTodo(n, locked)
		}
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			node.Content[i] = markLockedValuesAsTodo(node.Content[i], locked)
		}
	}

	return node
}

func isLocked(node *yaml.Node) bool {
	return node != nil && node.Tag == "!lock"
}

func isLocal(node *yaml.Node) bool {
	return node != nil && node.Tag == "!local"
}

type KeyValuePair struct {
	Key   *yaml.Node
	Value *yaml.Node
}

func asMap(node *yaml.Node) map[string]KeyValuePair {
	result := make(map[string]KeyValuePair, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		result[node.Content[i].Value] = KeyValuePair{
			Key:   node.Content[i],
			Value: node.Content[i+1],
		}
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

func mergeStyle(dst, src yaml.Style) yaml.Style {
	if src&yaml.FlowStyle == 0 {
		dst &^= yaml.FlowStyle
	}
	return dst
}

func firstNonNil(nodes ...*yaml.Node) *yaml.Node {
	for _, node := range nodes {
		if node != nil {
			return node
		}
	}
	return nil
}
