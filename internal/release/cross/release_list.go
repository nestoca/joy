package cross

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release"
	"path/filepath"
	"sort"
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
func Load(baseDir string, environments []*environment.Environment, releaseFilter release.Filter) (*ReleaseList, error) {
	releases := NewReleaseList(environments)
	for _, env := range environments {
		envDir := filepath.Join(baseDir, env.Name)
		envReleases, err := release.LoadAllInDir(envDir, releaseFilter)
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

func (r *ReleaseList) SortedReleases() []*Release {
	var releases []*Release
	for _, rel := range r.Releases {
		releases = append(releases, rel)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Name < releases[j].Name
	})
	return releases
}

func (r *ReleaseList) Subset(releases []string) *ReleaseList {
	subset := NewReleaseList(r.Environments)
	for _, rel := range releases {
		subset.Releases[rel] = r.Releases[rel]
	}
	return subset
}
