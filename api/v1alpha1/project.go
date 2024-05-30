package v1alpha1

import (
	"fmt"

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
	// It can be any string that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty"`

	// Reviewers is the list of GitHub users who should always added a reviewer for the project.
	Reviewers []string `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`

	// Git repository of the project.
	Repository string `yaml:"repository,omitempty" json:"repository,omitempty"`

	// Location of the project files in the repository. Should be empty if the whole repository is the project.
	// If there is more than one location, specify the main subdirectory of the project first.
	RepositorySubpaths []string `yaml:"repositorySubpaths,omitempty" json:"repositorySubpaths,omitempty"`

	// GitTagTemplate allows you to configure what your git tag look like relative to a release via go templates
	// example: gitTagTemplate: api/v{{ .Release.Spec.Version }}
	GitTagTemplate string `yaml:"gitTagTemplate,omitempty" json:"gitTagTemplate,omitempty"`

	// Links is the map of project-level overrides and additions for project links defined in catalog configuration.
	Links map[string]string `yaml:"links,omitempty" json:"links,omitempty"`

	// ReleaseLinks is the map of project-level overrides and additions for release links defined in catalog configuration.
	ReleaseLinks map[string]string `yaml:"releaseLinks,omitempty" json:"releaseLinks,omitempty"`
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
	if err := yml.UnmarshalStrict(file.Yaml, &proj); err != nil {
		return nil, fmt.Errorf("unmarshalling project: %w", err)
	}
	proj.File = file
	return &proj, nil
}
