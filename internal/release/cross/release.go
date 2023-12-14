package cross

import (
	"fmt"
	"path/filepath"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/style"
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
func (r *Release) ComputePromotedFile(sourceEnv, targetEnv *v1alpha1.Environment) error {
	sourceRelease := r.Releases[0]
	targetRelease := r.Releases[1]

	// Skip missing source releases, which obviously cannot be promoted
	if sourceRelease == nil {
		return nil
	}

	// Do we have an existing target release?
	var promotedFile *yml.File
	var err error
	if targetRelease != nil && targetRelease.File != nil {
		// Promote source release to existing target
		mergedTree := yml.Merge(targetRelease.File.Tree, sourceRelease.File.Tree)
		promotedFile, err = targetRelease.File.CopyWithNewTree(mergedTree)
		if err != nil {
			return fmt.Errorf("creating in-memory copy of target file using merged result: %w", err)
		}
	} else {
		relativePath, err := filepath.Rel(sourceEnv.Dir, sourceRelease.File.Path)
		if err != nil {
			return fmt.Errorf("failed to get promoted file's relative path within environment: %w", err)
		}

		// Promote source release to empty target
		promoted := yml.Merge(nil, sourceRelease.File.Tree)
		targetPath := filepath.Join(targetEnv.Dir, relativePath)

		promotedFile, err = yml.NewFileFromTree(targetPath, sourceRelease.File.Indent, promoted)
		if err != nil {
			return fmt.Errorf("creating in-memory file from tree for missing target release: %w", err)
		}
	}

	// Only consider promotion if the new merged result is different from existing target
	if targetRelease == nil || !yml.Compare(targetRelease.File.Yaml, promotedFile.Yaml) {
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
			r.Releases[i].Spec.Version != r.Releases[i+1].Spec.Version {
			return false
		}
	}
	return true
}

func GetReleaseDisplayVersion(rel *v1alpha1.Release, inSync bool) string {
	if rel == nil {
		return style.ReleaseNotAvailable("-")
	}
	if rel.Spec.Version == "" {
		return style.ReleaseInSyncOrNot("n/a", inSync)
	}
	return style.ReleaseInSyncOrNot(rel.Spec.Version, inSync)
}
