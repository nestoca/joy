package v1alpha1

import (
	"fmt"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/yaml.v3"
)

const ReleaseKind = "Release"

type ReleaseMetadata struct {
	// Name is the name of the release.
	Name string `yaml:"name,omitempty"`
}

type ReleaseChart struct {
	// Version of the chart.
	Version string `yaml:"version,omitempty"`
}

type ReleaseSpec struct {
	// Project is the name of the project that the release belongs to.
	Project string `yaml:"project,omitempty"`

	// Version of the release, typically corresponding to the image build version being deployed.
	Version string `yaml:"version,omitempty"`

	// Chart is the chart that the release is based on.
	Chart ReleaseChart `yaml:"chart,omitempty"`
}

type Release struct {
	// ApiVersion is the API version of the release.
	ApiVersion string `yaml:"apiVersion,omitempty"`

	// Kind is the kind of the release.
	Kind string `yaml:"kind,omitempty"`

	// ReleaseMetadata is the metadata of the release.
	ReleaseMetadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the release.
	Spec ReleaseSpec `yaml:"spec,omitempty"`

	// File represents the in-memory yaml file of the release.
	File *yml.File `yaml:"-"`

	// Missing indicates whether the release is missing in the target environment. During a promotion,
	// this allows to know whether the release will be created or updated.
	Missing bool `yaml:"-"`

	// Project is the project that the release belongs to.
	Project *Project `yaml:"-"`

	// Environment is the environment that the release is deployed to.
	Environment *Environment `yaml:"-"`
}

func IsValidRelease(apiVersion, kind string) bool {
	return apiVersion == "joy.nesto.ca/v1alpha1" && kind == ReleaseKind
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
