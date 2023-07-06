package promote

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release"
	"github.com/nestoca/joy-cli/internal/yml"
	"path/filepath"
)

// CreateMissingTargetReleases creates releases in target environment for releases that are in source environment but not in target.
func CreateMissingTargetReleases(crossReleases *release.CrossReleaseList) error {
	// Ensure we have two environments
	if len(crossReleases.Environments) != 2 {
		return fmt.Errorf("expected two environments, got %d", len(crossReleases.Environments))
	}
	targetEnv := crossReleases.Environments[1]

	// Iterate through promotable releases and create missing ones
	for _, crossRelease := range crossReleases.Releases {
		if crossRelease.Promotable() && crossRelease.Releases[1] == nil {
			// Create release in target environment
			srcRel := crossRelease.Releases[0]
			targetRel := createMissingRelease(srcRel, targetEnv)
			crossRelease.Releases[1] = targetRel
		}
	}
	return nil
}

func createMissingRelease(source *release.Release, env *environment.Environment) *release.Release {
	target := *source
	target.ReleaseFile.FilePath = filepath.Join(env.Dir, "releases", source.Name+".release.yaml")
	target.ValuesFile.FilePath = filepath.Join(env.Dir, "releases", source.Name+".values.yaml")
	target.ReleaseFile.Tree = yml.Merge(source.ReleaseFile.Tree, nil)
	target.ValuesFile.Tree = yml.Merge(source.ValuesFile.Tree, nil)
	target.Missing = true
	return &target
}
