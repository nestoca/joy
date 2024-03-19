package v1alpha1

import (
	"fmt"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/davidmdm/x/xerr"
	"github.com/nestoca/joy/internal/yml"
)

const EnvironmentKind = "Environment"

type EnvironmentMetadata struct {
	// Name is the name of the environment.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Labels is the list of labels for the environment.
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations is the list of annotations for the environment.
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type Promotion struct {
	AllowAutoMerge   bool     `yaml:"allowAutoMerge,omitempty" json:"allowAutoMerge,omitempty"`
	FromPullRequests bool     `yaml:"fromPullRequests,omitempty" json:"fromPullRequests,omitempty"`
	FromEnvironments []string `yaml:"fromEnvironments,omitempty" json:"fromEnvironments,omitempty"`
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

	// ChartVersions allows the environment to override the given version of the catalogs chart references.
	// This allows for environments to rollout new versions of chart references.
	ChartVersions map[string]string `yaml:"chartVersions,omitempty" json:"chartVersions,omitempty"`

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

func (env Environment) Validate(validChartRefs []string) error {
	var errs []error
	for ref := range env.Spec.ChartVersions {
		if !slices.Contains(validChartRefs, ref) {
			errs = append(errs, fmt.Errorf("unkown ref: %s", ref))
		}
	}
	return xerr.MultiErrOrderedFrom("validating chart references", errs...)
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

func GetEnvironmentNames(environments []*Environment) []string {
	var names []string
	for _, env := range environments {
		names = append(names, env.Name)
	}
	return names
}

func GetEnvironmentByName(environments []*Environment, name string) (*Environment, error) {
	if name == "" {
		return nil, nil
	}
	for _, env := range environments {
		if env.Name == name {
			return env, nil
		}
	}
	return nil, fmt.Errorf("environment %q not found", name)
}

// GetEnvironmentsByNames returns the subset of environments with given names, preserving
// their order in the original list. If names is empty, all environments are returned.
func GetEnvironmentsByNames(environments []*Environment, names []string) []*Environment {
	if len(names) == 0 {
		return environments
	}
	var envs []*Environment
	for _, env := range environments {
		if slices.Contains(names, env.Name) {
			envs = append(envs, env)
		}
	}
	return envs
}

func (e *Environment) IsPromotableTo(targetEnv *Environment) bool {
	for _, source := range targetEnv.Spec.Promotion.FromEnvironments {
		if source == e.Name {
			return true
		}
	}
	return false
}
