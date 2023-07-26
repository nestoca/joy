package release

import (
	"fmt"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/project"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/yaml.v3"
)

const Kind = "Release"

type Metadata struct {
	// Name is the name of the release.
	Name string `yaml:"name,omitempty"`
}

type Chart struct {
	// Version of the chart.
	Version string `yaml:"version,omitempty"`
}

type Spec struct {
	// Project is the name of the project that the release belongs to.
	Project string `yaml:"project,omitempty"`

	// Version of the release, typically corresponding to the image build version being deployed.
	Version string `yaml:"version,omitempty"`

	// Chart is the chart that the release is based on.
	Chart Chart `yaml:"chart,omitempty"`
}

type Release struct {
	// ApiVersion is the API version of the release.
	ApiVersion string `yaml:"apiVersion,omitempty"`

	// Kind is the kind of the release.
	Kind string `yaml:"kind,omitempty"`

	// Metadata is the metadata of the release.
	Metadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the release.
	Spec Spec `yaml:"spec,omitempty"`

	// File represents the in-memory yaml file of the release.
	File *yml.File `yaml:"-"`

	// Missing indicates whether the release is missing in the target environment. During a promotion,
	// this allows to know whether the release will be created or updated.
	Missing bool `yaml:"-"`

	// Project is the project that the release belongs to.
	Project *project.Project `yaml:"-"`

	// Environment is the environment that the release is deployed to.
	Environment *environment.Environment `yaml:"-"`
}

func IsValid(apiVersion, kind string) bool {
	return apiVersion == "joy.nesto.ca/v1alpha1" && kind == Kind
}

// LoadRelease loads a release from the given release file.
func LoadRelease(file *yml.File) (*Release, error) {
	var rel Release
	if err := yaml.Unmarshal(file.Yaml, &rel); err != nil {
		return nil, fmt.Errorf("unmarshalling release: %w", err)
	}
	rel.File = file
	return &rel, nil
}
