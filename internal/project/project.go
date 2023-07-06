package project

import "github.com/nestoca/joy-cli/internal/yml"

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

	// Missing indicates whether the project file is missing in the projects directory.
	Missing bool `yaml:"-"`
}
