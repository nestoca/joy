package releasing

import (
	"github.com/nestoca/joy-cli/internal/environment"
)

// CrossRelease describes a given release across multiple environments
type CrossRelease struct {
	// Name is the name of the release.
	Name string

	// Releases is the list of releases for a given release name across multiple environments.
	Releases []*Release
}

func NewCrossRelease(name string, environments []*environment.Environment) *CrossRelease {
	return &CrossRelease{
		Name:     name,
		Releases: make([]*Release, len(environments)),
	}
}

// AllReleasesSynced returns true if all releases are synced across all environments.
func (r *CrossRelease) AllReleasesSynced() bool {
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

// AllValuesSynced returns true if all values are synced across all environments.
func (r *CrossRelease) AllValuesSynced() bool {
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
