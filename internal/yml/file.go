package yml

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File represents a yaml file loaded into memory in different forms,
// that can be round-tripped back to disk.
type File struct {
	// Path is the path to the yaml file.
	Path string

	// Yaml is the raw yaml of the yaml file.
	Yaml []byte

	// Tree is the root node of the tree representation of the yaml file.
	Tree *yaml.Node

	// ApiVersion is the API version of the CRD, if any.
	ApiVersion string

	// Kind is the kind of the CRD, if any.
	Kind string

	// MetadataName is the name of the CRD resource, if any.
	MetadataName string

	// Indent is the indent size of the yaml file to be used when writing it back to disk.
	Indent int
}

func NewFile(filePath string, content []byte) (*File, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s in yaml node form: %w", filePath, err)
	}

	cleanFilePath, err := cleanUpFilePath(filePath)
	if err != nil {
		return nil, err
	}

	return &File{
		Path:         cleanFilePath,
		Yaml:         content,
		Tree:         &node,
		ApiVersion:   FindNodeValueOrDefault(&node, "apiVersion", ""),
		Kind:         FindNodeValueOrDefault(&node, "kind", ""),
		MetadataName: FindNodeValueOrDefault(&node, "metadata.name", ""),
		Indent:       getIndentSize(string(content)),
	}, nil
}

func NewFileFromTree(filePath string, indent int, node *yaml.Node) (*File, error) {
	content, err := marshallTreeToYaml(node, indent)
	if err != nil {
		return nil, fmt.Errorf("marshalling yaml node to yaml: %w", err)
	}

	cleanFilePath, err := cleanUpFilePath(filePath)
	if err != nil {
		return nil, err
	}

	return &File{
		Path:         cleanFilePath,
		Yaml:         content,
		Tree:         node,
		ApiVersion:   FindNodeValueOrDefault(node, "apiVersion", ""),
		Kind:         FindNodeValueOrDefault(node, "kind", ""),
		MetadataName: FindNodeValueOrDefault(node, "metadata.name", ""),
		Indent:       indent,
	}, nil
}

func cleanUpFilePath(filePath string) (string, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path of %s: %w", filePath, err)
	}
	cleanFilePath := filepath.Clean(absFilePath)
	return cleanFilePath, nil
}

func LoadFile(filePath string) (*File, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading yaml file %s: %w", filePath, err)
	}
	return NewFile(filePath, content)
}

func (y *File) CopyWithNewTree(newTree *yaml.Node) (*File, error) {
	newYaml, err := marshallTreeToYaml(newTree, y.Indent)
	if err != nil {
		return nil, err
	}

	return &File{
		Path:         y.Path,
		Yaml:         newYaml,
		Tree:         newTree,
		ApiVersion:   y.ApiVersion,
		Kind:         y.Kind,
		MetadataName: y.MetadataName,
		Indent:       y.Indent,
	}, nil
}

func (y *File) UpdateYamlFromTree() error {
	newYaml, err := marshallTreeToYaml(y.Tree, y.Indent)
	if err != nil {
		return err
	}
	y.Yaml = newYaml
	return nil
}

func marshallTreeToYaml(tree *yaml.Node, indent int) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(indent)
	err := encoder.Encode(tree)
	if err != nil {
		return nil, fmt.Errorf("marshalling node tree to yaml: %w", err)
	}
	return buf.Bytes(), nil
}

func (y *File) WriteYaml() error {
	return os.WriteFile(y.Path, y.Yaml, 0o644)
}

func (y *File) ToYaml() (string, error) {
	return TreeToYaml(y.Tree, y.Indent)
}
