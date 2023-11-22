package v1alpha1

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

const ProjectKind = "Project"

type ProjectMetadata struct {
	// Name is the name of the project.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Labels is the list of labels for the project.
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations is the list of annotations for the project.
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type ProjectSpec struct {
	// Owners is the list of identifiers of owners of the project.
	// It can be any strings that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty"`

	// Git repository of the project.
	Repository string `yaml:"repository,omitempty" json:"repository,omitempty"`

	// Location of the project in the repository. Should be empty if the whole repository is the project.
	RepositorySubdirectory string `yaml:"repositorySubdirectory,omitempty" json:"repositorySubdirectory,omitempty"`
}

type Project struct {
	// ApiVersion is the API version of the project.
	ApiVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Kind is the kind of the project.
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// ProjectMetadata is the metadata of the project.
	ProjectMetadata `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	// Spec is the spec of the project.
	Spec ProjectSpec `yaml:"spec,omitempty" json:"spec,omitempty"`

	// File represents the in-memory yaml file of the project.
	File *yml.File `yaml:"-" json:"-"`
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
