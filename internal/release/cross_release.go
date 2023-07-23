package release

import (
	"github.com/nestoca/joy/internal/environment"
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
		if rel == nil || rel.Missing {
			return false
		}
		if hash == 0 {
			hash = rel.File.Hash
		} else if rel.File.Hash != hash {
			return false
		}
	}
	return true
}

// Promotable returns whether release can be promoted. In other words, that it has a release in the first environment.
// This assumes that there are two environments and that first one is the source and the second one is the target.
func (r *CrossRelease) Promotable() bool {
	return len(r.Releases) == 2 && r.Releases[0] != nil
}
