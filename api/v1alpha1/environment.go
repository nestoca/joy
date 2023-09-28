package v1alpha1

import (
	"fmt"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/yaml.v3"
	"path/filepath"
)

const EnvironmentKind = "Environment"

type EnvironmentMetadata struct {
	// Name is the name of the environment.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

type Promotion struct {
	FromPullRequests bool `yaml:"fromPullRequests,omitempty" json:"fromPullRequests,omitempty"`
}

type EnvironmentSpec struct {
	// Order controls the display order of the environment.
	Order int `yaml:"order,omitempty" json:"order,omitempty"`

	// Promotion controls the promotion of releases to this environment.
	Promotion Promotion `yaml:"promotion,omitempty" json:"promotion,omitempty"`

	// Cluster is the name of environment's cluster.
	Cluster string `yaml:"cluster,omitempty" json:"cluster,omitempty"`

	// Namespace is the name of environment's namespace within cluster.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Owners is the list of identifiers of owners of the environment.
	// It can be any strings that uniquely identifies the owners, such as email addresses or Jac group identifiers.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty"`

	// SealedSecretsCert is the public certificate of the Sealed Secrets controller for this environment
	// that can be used to encrypt secrets targeted to this environment using the `joy secret seal` command.
	SealedSecretsCert string `yaml:"sealedSecretsCert,omitempty" json:"sealedSecretsCert,omitempty"`
}

type Environment struct {
	// ApiVersion is the API version of the environment.
	ApiVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Kind is the kind of the environment.
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// EnvironmentMetadata is the metadata of the environment.
	EnvironmentMetadata `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	// Spec is the spec of the environment.
	Spec EnvironmentSpec `yaml:"spec,omitempty" json:"spec,omitempty"`

	// File represents the in-memory yaml file of the project.
	File *yml.File `yaml:"-" json:"-"`

	// Dir is the path to the environment directory.
	Dir string `yaml:"-" json:"-"`
}

func IsValidEnvironment(apiVersion, kind string) bool {
	return apiVersion == "joy.nesto.ca/v1alpha1" && kind == EnvironmentKind
}

// NewEnvironment creates a new environment from given yaml file.
func NewEnvironment(file *yml.File) (*Environment, error) {
	var env Environment
	if err := yaml.Unmarshal(file.Yaml, &env); err != nil {
		return nil, fmt.Errorf("unmarshalling environment: %w", err)
	}
	env.File = file
	env.Dir = filepath.Dir(file.Path)
	return &env, nil
}
