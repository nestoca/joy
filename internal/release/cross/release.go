package cross

import (
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release"
)

// Release describes a given release across multiple environments
type Release struct {
	// Name is the name of the release.
	Name string

	// Releases is the list of releases for a given release name across multiple environments.
	Releases []*release.Release
}

func NewRelease(name string, environments []*environment.Environment) *Release {
	return &Release{
		Name:     name,
		Releases: make([]*release.Release, len(environments)),
	}
}

// AreReleasesSynced returns true if all releases are synced across all environments.
func (r *Release) AreReleasesSynced() bool {
	var hash uint64
	for _, rel := range r.Releases {
		if rel == nil {
			return false
		}
		if hash == 0 {
			hash = rel.ReleaseFile.Hash
		} else if rel.ReleaseFile.Hash != hash {
			return false
		}
	}
	return true
}

// AreValuesSynced returns true if all values are synced across all environments.
func (r *Release) AreValuesSynced() bool {
	var hash uint64
	for _, rel := range r.Releases {
		if rel == nil {
			return false
		}
		if hash == 0 {
			hash = rel.ValuesFile.Hash
		} else if rel.ValuesFile.Hash != hash {
			return false
		}
	}
	return true
}
