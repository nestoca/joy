package environment

import (
	"fmt"
	"os"
	"path/filepath"
)

type Environment struct {
	// Name is the name identifier of the environment (eg: dev, staging, prod).
	Name string

	// Dir is the path to the environment file.
	Dir string

	// Order controls the display order of the environment.
	Order int
}

// New creates a new environment.
func New(name string) *Environment {
	return &Environment{
		Name: name,
	}
}

func NewList(names []string) []*Environment {
	var environments []*Environment
	for _, name := range names {
		environments = append(environments, New(name))
	}
	return environments
}

func LoadAll(baseDir string) ([]*Environment, error) {
	// Ensure dir exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", baseDir)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", baseDir, err)
	}

	var environments []*Environment
	for _, entry := range entries {
		if entry.IsDir() {
			dir := filepath.Join(baseDir, entry.Name())
			environment, err := Load(dir)
			if err != nil {
				return nil, fmt.Errorf("loading environment from %q: %w", dir, err)
			}
			environments = append(environments, environment)
		}
	}

	return environments, nil
}

func Load(dir string) (*Environment, error) {
	environment := New(filepath.Base(dir))
	environment.Dir = dir
	return environment, nil
}
