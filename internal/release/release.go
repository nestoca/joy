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
	Name string
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
	Metadata Metadata `yaml:"metadata,omitempty"`

	// Spec is the spec of the release.
	Spec Spec `yaml:"spec,omitempty"`

	// FilePath is the path to the release file.
	FilePath string `yaml:"-"`

	// Yaml is the raw yaml of the release.
	Yaml string `yaml:"-"`
}

// LoadAllInDir loads all releases in the given directory.
func LoadAllInDir(dir string) ([]*Release, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var releases []*Release
	for _, file := range files {
		fileName := file.Name()
		if strings.HasSuffix(fileName, ".release.yaml") {
			filePath := filepath.Join(dir, fileName)
			release, err := Load(filePath)
			if err != nil {
				return nil, fmt.Errorf("loading release %s: %w", filePath, err)
			}
			releases = append(releases, release)
		}
	}

	return releases, nil
}

// Load loads a release from the given release file.
func Load(releaseFile string) (*Release, error) {
	content, err := os.ReadFile(releaseFile)
	if err != nil {
		return nil, fmt.Errorf("reading release file %s: %w", releaseFile, err)
	}

	var release Release
	if err := yaml.Unmarshal(content, &release); err != nil {
		return nil, fmt.Errorf("unmarshalling release file %s: %w", releaseFile, err)
	}
	release.FilePath = releaseFile
	release.Yaml = string(content)

	return &release, nil
}
