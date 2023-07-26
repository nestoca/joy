package cross

import (
	"github.com/nestoca/joy/api/v1alpha1"
)

// Release describes a given release across multiple environments
type Release struct {
	// Name is the name of the release.
	Name string

	// Releases is the list of releases for a given release name across multiple environments.
	Releases []*v1alpha1.Release
}

func NewRelease(name string, environments []*v1alpha1.Environment) *Release {
	return &Release{
		Name:     name,
		Releases: make([]*v1alpha1.Release, len(environments)),
	}
}

// AllReleasesSynced returns true if all releases are synced across all environments.
func (r *Release) AllReleasesSynced() bool {
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
// This assumes that there are two and only two environments and that first one is the source and the second one is the
// target. Promotability is defined as having the release present at least in source environment.
func (r *Release) Promotable() bool {
	return len(r.Releases) == 2 && r.Releases[0] != nil
}
