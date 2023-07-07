package environment

import (
	"fmt"
	"github.com/nestoca/joy/internal/yml"
	"os"
	"path/filepath"
)

const DirName = "environments"

type Metadata struct {
	// Name is the name of the environment.
	Name string `yaml:"name,omitempty"`
}

type Spec struct {
	// Order controls the display order of the environment.
	Order int `yaml:"order,omitempty"`

	// Owners is the list of identifiers of owners of the environment.
	// It can be any strings that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty"`
}

type Environment struct {
	// ApiVersion is the API version of the environment.
	ApiVersion string `yaml:"apiVersion,omitempty"`

	// Kind is the kind of the environment.
	Kind string `yaml:"kind,omitempty"`

	// Metadata is the metadata of the environment.
	Metadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the environment.
	Spec Spec `yaml:"spec,omitempty"`

	// File represents the in-memory yaml file of the project.
	File *yml.File `yaml:"-"`

	// Missing indicates whether the environment file is missing in the environment directory.
	Missing bool `yaml:"-"`

	// Dir is the path to the environment directory.
	Dir string `yaml:"-"`
}

// New creates a new environment.
func New(name string) *Environment {
	return &Environment{
		Metadata: Metadata{Name: name},
	}
}

func LoadAll(baseDir string, names ...string) ([]*Environment, error) {
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
			// Skip if not in names
			envName := entry.Name()
			if len(names) > 0 {
				found := false
				for _, name := range names {
					if name == envName {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Load environment
			dir := filepath.Join(baseDir, envName)
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
