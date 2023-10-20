package cross

import (
	"fmt"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/yml"
)

// Release describes a given release across multiple environments
type Release struct {
	// Name is the name of the release.
	Name string

	// Releases is the list of releases for a given release name across multiple environments.
	Releases []*v1alpha1.Release

	// PromotedFile is the merged file for release from source to target environment, assuming the only environments
	// are respectively source and target. If merged result is same as target, then no promotion is needed and
	// PromotedFile is nil. This must be explicitly computed via ComputePromotedFile().
	PromotedFile *yml.File
}

// ComputePromotedFile computes the promotion merged file for release from source to target environment,
// assuming the only environments are respectively source and target. If merged result is same as target,
// then no promotion is needed and PromotedFile is nil.
func (r *Release) ComputePromotedFile() error {
	source := r.Releases[0].File
	target := r.Releases[1].File
	mergedTree := yml.Merge(source.Tree, target.Tree)
	promotedFile, err := target.CopyWithNewTree(mergedTree)
	if err != nil {
		return fmt.Errorf("making in-memory copy of target file using merged result: %w", err)
	}
	if !yml.Compare(target.Yaml, promotedFile.Yaml) {
		r.PromotedFile = promotedFile
	} else {
		r.PromotedFile = nil
	}
	return nil
}

func NewRelease(name string, environments []*v1alpha1.Environment) *Release {
	return &Release{
		Name:     name,
		Releases: make([]*v1alpha1.Release, len(environments)),
	}
}

func (r *Release) AreVersionsInSync() bool {
	for i := 0; i < len(r.Releases)-1; i++ {
		if r.Releases[i] == nil ||
			r.Releases[i+1] == nil ||
			r.Releases[i].Missing ||
			r.Releases[i+1].Missing ||
			r.Releases[i].Spec.Version != r.Releases[i+1].Spec.Version {
			return false
		}
	}
	return true
}

