package release

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

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

	// ReleaseFile represents the in-memory yaml file of the release.
	ReleaseFile *YamlFile `yaml:"-"`

	// ValuesFile represents the in-memory yaml file of the values associated with the release.
	ValuesFile *YamlFile `yaml:"-"`

	// Missing indicates whether the release is missing in the target environment. During a promotion,
	// this allows to know whether the release will be created or updated.
	Missing bool `yaml:"-"`
}

// LoadAllInDir loads all releases in the given directory.
func LoadAllInDir(dir string, releaseFilter Filter) ([]*Release, error) {
	dir = filepath.Join(dir, "releases")

	// Ensure dir exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", dir)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var releases []*Release
	for _, file := range files {
		fileName := file.Name()
		if strings.HasSuffix(fileName, ".release.yaml") {
			filePath := filepath.Join(dir, fileName)
			rel, err := LoadRelease(filePath)
			if err != nil {
				return nil, fmt.Errorf("loading release %s: %w", filePath, err)
			}

			if releaseFilter == nil || releaseFilter.Match(rel) {
				releases = append(releases, rel)
			}
		}
	}

	return releases, nil
}

// LoadRelease loads a release from the given release file.
func LoadRelease(filePath string) (*Release, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading release file %s: %w", filePath, err)
	}

	// Load in structured form
	var rel Release
	if err := yaml.Unmarshal(content, &rel); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s in structured form: %w", filePath, err)
	}

	// Load in yaml file form
	yamlFile, err := NewYamlFile(filePath, content)
	if err != nil {
		return nil, fmt.Errorf("creating yaml file for release file %s: %w", filePath, err)
	}
	rel.ReleaseFile = yamlFile

	// Load values file
	valuesFilePath := strings.TrimSuffix(filePath, ".release.yaml") + ".values.yaml"
	valuesContent, err := os.ReadFile(valuesFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading values file %s: %w", valuesFilePath, err)
	}
	valuesYamlFile, err := NewYamlFile(valuesFilePath, valuesContent)
	if err != nil {
		return nil, fmt.Errorf("creating yaml file for values file %s: %w", valuesFilePath, err)
	}
	rel.ValuesFile = valuesYamlFile

	return &rel, nil
}
