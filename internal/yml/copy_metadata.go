package yml

import (
	"slices"

	"gopkg.in/yaml.v3"
)

// CopyMetadata reconciles dst to the authored form of src, restoring presentation that is lost
// when a tree is rebuilt from JSON (e.g. after applying a JSON Patch). It copies, onto matching
// nodes:
//
//   - custom YAML tags (those in CustomTags, e.g. !lock),
//   - head/line/foot comments,
//   - and, for mappings, src's key order (keys present only in dst — e.g. added by a patch — are
//     kept and appended after the src-ordered ones).
//
// It walks both trees in parallel, stopping on any branch where the nodes are nil or differ in
// kind (i.e. structurally changed relative to src). It never moves data between branches: every
// dst node is preserved.
func CopyMetadata(dst, src *yaml.Node) {
	if dst == nil || src == nil || dst.Kind != src.Kind {
		return
	}

	if slices.Contains(CustomTags, src.Tag) {
		dst.Tag = src.Tag
	}
	dst.HeadComment = src.HeadComment
	dst.LineComment = src.LineComment
	dst.FootComment = src.FootComment

	switch dst.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for i := 0; i < min(len(dst.Content), len(src.Content)); i++ {
			CopyMetadata(dst.Content[i], src.Content[i])
		}
	case yaml.MappingNode:
		dstByKey := asMap(dst)
		reordered := make([]*yaml.Node, 0, len(dst.Content))
		seen := make(map[string]bool, len(dst.Content)/2)

		// dst pairs in src's key order, recursing into the matched ones.
		for i := 0; i+1 < len(src.Content); i += 2 {
			key := src.Content[i].Value
			dstPair, ok := dstByKey[key]
			if !ok {
				continue
			}
			CopyMetadata(dstPair.Key, src.Content[i])
			CopyMetadata(dstPair.Value, src.Content[i+1])
			reordered = append(reordered, dstPair.Key, dstPair.Value)
			seen[key] = true
		}
		// dst-only keys (e.g. added by a patch), in their current order.
		for i := 0; i+1 < len(dst.Content); i += 2 {
			if key := dst.Content[i].Value; !seen[key] {
				reordered = append(reordered, dst.Content[i], dst.Content[i+1])
			}
		}
		dst.Content = reordered
	}
}
