package project

import (
	"fmt"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/yaml.v3"
)

const Kind = "Project"

type Metadata struct {
	// Name is the name of the project.
	Name string `yaml:"name,omitempty"`
}

type Spec struct {
	// Owners is the list of identifiers of owners of the project.
	// It can be any strings that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty"`
}

type Project struct {
	// ApiVersion is the API version of the project.
	ApiVersion string `yaml:"apiVersion,omitempty"`

	// Kind is the kind of the project.
	Kind string `yaml:"kind,omitempty"`

	// Metadata is the metadata of the project.
	Metadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the project.
	Spec Spec `yaml:"spec,omitempty"`

	// File represents the in-memory yaml file of the project.
	File *yml.File `yaml:"-"`
}

func IsValid(apiVersion, kind string) bool {
	return apiVersion == "joy.nesto.ca/v1alpha1" && kind == Kind
}

// New creates a new project from given yaml file.
func New(file *yml.File) (*Project, error) {
	var proj Project
	if err := yaml.Unmarshal(file.Yaml, &proj); err != nil {
		return nil, fmt.Errorf("unmarshalling project: %w", err)
	}
	proj.File = file
	return &proj, nil
}
