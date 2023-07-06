package release

import (
	"gopkg.in/yaml.v3"
	"hash"
	"hash/fnv"
)

func GetHash(node *yaml.Node) uint64 {
	hsh := fnv.New64a()
	getHashRecursively(node, hsh)
	return hsh.Sum64()
}

func getHashRecursively(node *yaml.Node, hsh hash.Hash64) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		getHashRecursively(node.Content[0], hsh)
	case yaml.MappingNode:
		_, _ = hsh.Write([]byte("mapping"))
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if IsLocked(key, value) {
				continue
			}
			getHashRecursively(key, hsh)
			getHashRecursively(value, hsh)
		}
	case yaml.SequenceNode:
		_, _ = hsh.Write([]byte("sequence"))
		for _, item := range node.Content {
			getHashRecursively(item, hsh)
		}
	case yaml.ScalarNode:
		_, _ = hsh.Write([]byte(node.Value))
	}
}
