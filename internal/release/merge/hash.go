package merge

import (
	"gopkg.in/yaml.v3"
	"hash"
	"hash/fnv"
)

func GetHash(node *yaml.Node) uint64 {
	hsh := fnv.New64a()
	traverse(node, hsh)
	return hsh.Sum64()
}

func traverse(node *yaml.Node, hsh hash.Hash64) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		traverse(node.Content[0], hsh)
	case yaml.MappingNode:
		_, _ = hsh.Write([]byte("mapping"))
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if isLockedSubtree(key, value) {
				continue
			}
			traverse(key, hsh)
			traverse(value, hsh)
		}
	case yaml.SequenceNode:
		_, _ = hsh.Write([]byte("sequence"))
		for _, item := range node.Content {
			traverse(item, hsh)
		}
	case yaml.ScalarNode:
		_, _ = hsh.Write([]byte(node.Value))
	}
}
