package release

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// YamlFile represents a yaml file loaded into memory in different forms,
// that can be round-tripped back to disk.
type YamlFile struct {
	// FilePath is the path to the values file.
	FilePath string `yaml:"-"`

	// Yaml is the raw yaml of the values file.
	Yaml []byte `yaml:"-"`

	// Node is the yaml root node of the values file.
	Node *yaml.Node `yaml:"-"`

	// Hash is the hash of the values file.
	Hash uint64 `yaml:"-"`
}

func NewYamlFile(filePath string, content []byte) (*YamlFile, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s in yaml node form: %w", filePath, err)
	}



	return &YamlFile{
		FilePath: filePath,
		Yaml:     content,
		Node:     &node,
	}, nil
}
