package yml

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// File represents a yaml file loaded into memory in different forms,
// that can be round-tripped back to disk.
type File struct {
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

func NewFile(filePath string, content []byte) (*File, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s in yaml node form: %w", filePath, err)
	}
	hash := GetHash(&node)
	return &File{
		FilePath: filePath,
		Yaml:     content,
		Tree:     &node,
		Hash:     hash,
		Indent:   getIndentSize(string(content)),
	}, nil
}

func (y *File) CopyWithNewTree(newTree *yaml.Node) (*File, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(y.Indent)
	err := encoder.Encode(newTree)
	if err != nil {
		return nil, fmt.Errorf("encoding node tree to yaml: %w", err)
	}

	return &File{
		FilePath: y.FilePath,
		Yaml:     buf.Bytes(),
		Tree:     newTree,
		Hash:     GetHash(newTree),
		Indent:   y.Indent,
	}, nil
}

func (y *File) Write() error {
	return os.WriteFile(y.FilePath, y.Yaml, 0644)
}

func (y *File) ToYaml() (string, error) {
	return TreeToYaml(y.Tree, y.Indent)
}
