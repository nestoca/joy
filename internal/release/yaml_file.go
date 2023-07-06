package release

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// YamlFile represents a yaml file loaded into memory in different forms,
// that can be round-tripped back to disk.
type YamlFile struct {
	// FilePath is the path to the yaml file.
	FilePath string

	// Yaml is the raw yaml of the yaml file.
	Yaml []byte

	// Tree is the root node of the tree representation of the yaml file.
	Tree *yaml.Node

	// Hash is the hash of the yaml file.
	Hash uint64

	// Indent is the indent size of the yaml file to be used when writing it back to disk.
	Indent int
}

func NewYamlFile(filePath string, content []byte) (*YamlFile, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s in yaml node form: %w", filePath, err)
	}
	hash := GetHash(&node)
	return &YamlFile{
		FilePath: filePath,
		Yaml:     content,
		Tree:     &node,
		Hash:     hash,
		Indent:   getIndentSize(string(content)),
	}, nil
}

func (y *YamlFile) CopyWithNewTree(newTree *yaml.Node) (*YamlFile, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(y.Indent)
	err := encoder.Encode(newTree)
	if err != nil {
		return nil, fmt.Errorf("encoding node tree to yaml: %w", err)
	}

	return &YamlFile{
		FilePath: y.FilePath,
		Yaml:     buf.Bytes(),
		Tree:     newTree,
		Hash:     GetHash(newTree),
		Indent:   y.Indent,
	}, nil
}

func (y *YamlFile) Write() error {
	return os.WriteFile(y.FilePath, y.Yaml, 0644)
}

func (y *YamlFile) ToYaml() (string, error) {
	return TreeToYaml(y.Tree, y.Indent)
}

func TreeToYaml(tree *yaml.Node, indent int) (string, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(indent)
	err := encoder.Encode(tree)
	if err != nil {
		return "", fmt.Errorf("encoding node tree to yaml: %w", err)
	}
	return buf.String(), nil
}
