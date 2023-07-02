package cross

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release"
	"path/filepath"
)

type ReleaseList struct {
	Environments []*environment.Environment
	Releases     map[string]*Release
}

// NewReleaseList creates a new ReleaseList
func NewReleaseList(environments []*environment.Environment) *ReleaseList {
	return &ReleaseList{
		Environments: environments,
		Releases:     make(map[string]*Release),
	}
}

// Load loads all releases for given environments underneath the given base directory.
func Load(baseDir string, environments []*environment.Environment) (*ReleaseList, error) {
	releases := NewReleaseList(environments)
	for _, env := range environments {
		envDir := filepath.Join(baseDir, env.Name)
		envReleases, err := release.LoadAllInDir(envDir)
		if err != nil {
			return nil, fmt.Errorf("loading releases in %s: %w", envDir, err)
		}
		for _, rel := range envReleases {
			err := releases.AddRelease(rel, env)
			if err != nil {
				return nil, fmt.Errorf("adding release %s: %w", rel.Name, err)
			}
		}
	}
	return releases, nil
}

// GetEnvironmentIndex returns the index of the environment with the given name or -1 if not found.
func (r *ReleaseList) GetEnvironmentIndex(name string) int {
	for i, env := range r.Environments {
		if env.Name == name {
			return i
		}
	}
	return -1
}

// AddRelease adds a release for given environment.
func (r *ReleaseList) AddRelease(release *release.Release, environment *environment.Environment) error {
	index := r.GetEnvironmentIndex(environment.Name)
	if index == -1 {
		return fmt.Errorf("environment %s not found in list", environment.Name)
	}

	rel, ok := r.Releases[release.Name]
	if !ok {
		rel = NewRelease(release.Name, r.Environments)
		r.Releases[release.Name] = rel
	}
	rel.Releases[index] = release
	return nil
}
