package v1alpha1

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

const ReleaseKind = "Release"

type ReleaseMetadata struct {
	// Name is the name of the release.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Labels is the list of labels for the release.
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations is the list of annotations for the release.
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type ReleaseChart struct {
	// Name is the name of the chart.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// RepoUrl is the url of the chart repository.
	RepoUrl string `yaml:"repoUrl,omitempty" json:"repoUrl,omitempty"`

	// Version of the chart.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type ReleaseSpec struct {
	// Project is the name of the project that the release belongs to.
	Project string `yaml:"project,omitempty" json:"project,omitempty"`

	// Version of the release, typically corresponding to the image build version being deployed.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Chart is the chart that the release is based on.
	Chart ReleaseChart `yaml:"chart,omitempty" json:"chart,omitempty"`

	// Values is the values to use to render the chart.
	Values map[string]interface{} `yaml:"values,omitempty" json:"values,omitempty"`
}

type Release struct {
	// ApiVersion is the API version of the release.
	ApiVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Kind is the kind of the release.
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// ReleaseMetadata is the metadata of the release.
	ReleaseMetadata `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	// Spec is the spec of the release.
	Spec ReleaseSpec `yaml:"spec,omitempty" json:"spec,omitempty"`

	// File represents the in-memory yaml file of the release.
	File *yml.File `yaml:"-" json:"-"`

	// Project is the project that the release belongs to.
	Project *Project `yaml:"-" json:"-"`

	// Environment is the environment that the release is deployed to.
	Environment *Environment `yaml:"-" json:"-"`
}

func (release *Release) UnmarshalYAML(node *yaml.Node) error {
	type raw Release
	return stripCustomTags(yml.Clone(node)).Decode((*raw)(release))
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

func stripCustomTags(node *yaml.Node) *yaml.Node {
	if slices.Contains(yml.CustomTags, node.Tag) {
		node.Tag = ""
	}
	for i, content := range node.Content {
		node.Content[i] = stripCustomTags(content)
	}
	return node
}
