package v1alpha1

import (
	"fmt"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/yaml.v3"
)

const ProjectKind = "Project"

type ProjectMetadata struct {
	// Name is the name of the project.
	Name string `yaml:"name,omitempty"`
}

type ProjectSpec struct {
	// Owners is the list of identifiers of owners of the project.
	// It can be any strings that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty"`
}

type Project struct {
	// ApiVersion is the API version of the project.
	ApiVersion string `yaml:"apiVersion,omitempty"`

	// Kind is the kind of the project.
	Kind string `yaml:"kind,omitempty"`

	// ProjectMetadata is the metadata of the project.
	ProjectMetadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the project.
	Spec ProjectSpec `yaml:"spec,omitempty"`

	// File represents the in-memory yaml file of the project.
	File *yml.File `yaml:"-"`
}

func IsValidProject(apiVersion, kind string) bool {
	return apiVersion == "joy.nesto.ca/v1alpha1" && kind == ProjectKind
}

// NewProject creates a new project from given yaml file.
func NewProject(file *yml.File) (*Project, error) {
	var proj Project
	if err := yaml.Unmarshal(file.Yaml, &proj); err != nil {
		return nil, fmt.Errorf("unmarshalling project: %w", err)
	}
	proj.File = file
	return &proj, nil
}
