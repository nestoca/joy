package yml

import (
	"bytes"
	"cmp"
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

func (file *File) Yaml() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(cmp.Or(file.Indent, 2))
	if err := encoder.Encode(file.Tree); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (file *File) MustYaml() []byte {
	data, err := file.Yaml()
	if err != nil {
		panic(err)
	}
	return data
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
		Tree:         &node,
		ApiVersion:   FindNodeValueOrDefault(&node, "apiVersion", ""),
		Kind:         FindNodeValueOrDefault(&node, "kind", ""),
		MetadataName: FindNodeValueOrDefault(&node, "metadata.name", ""),
		Indent:       getIndentSize(string(content)),
	}, nil
}

func NewFileFromTree(filePath string, indent int, node *yaml.Node) (*File, error) {
	cleanFilePath, err := cleanUpFilePath(filePath)
	if err != nil {
		return nil, err
	}

	return &File{
		Path:         cleanFilePath,
		Tree:         node,
		ApiVersion:   FindNodeValueOrDefault(node, "apiVersion", ""),
		Kind:         FindNodeValueOrDefault(node, "kind", ""),
		MetadataName: FindNodeValueOrDefault(node, "metadata.name", ""),
		Indent:       indent,
	}, nil
}

func NewFileFromObject(filePath string, indent int, obj any) (*File, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(indent)
	err := encoder.Encode(obj)
	if err != nil {
		return nil, fmt.Errorf("marshalling object to yaml: %w", err)
	}

	return NewFile(filePath, buf.Bytes())
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
	return &File{
		Path:         y.Path,
		Tree:         newTree,
		ApiVersion:   y.ApiVersion,
		Kind:         y.Kind,
		MetadataName: y.MetadataName,
		Indent:       y.Indent,
	}, nil
}
