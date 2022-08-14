package internal

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

type File struct {
	// path is the absolute path of file that was loaded and/or where to write it back.
	path string

	// document is the yaml node that was loaded from corresponding file.
	document yaml.Node
}

type Files []*File

func LoadFile(filePath string) (*File, error) {
	return nil, fmt.Errorf("not implemented")
}

func LoadFiles(dir string) (Files, error) {
	return nil, fmt.Errorf("not implemented")
}

func LoadFilesRecursively(dir string) (Files, error) {
	return nil, fmt.Errorf("not implemented")
}

func (*File) Save() error {
	return fmt.Errorf("not implemented")
}

func (Files) Save() error {
	return fmt.Errorf("not implemented")
}
