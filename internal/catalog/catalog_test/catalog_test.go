package catalog_test

import (
	"github.com/nestoca/joy/internal/catalog"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestFreeformEnvsAndReleasesLoading(t *testing.T) {
	catalogDir, err := filepath.Abs("testdata/freeform")
	assert.NoError(t, err)
	loadOpts := catalog.LoadOpts{
		Dir:             catalogDir,
		LoadEnvs:        true,
		LoadReleases:    true,
		LoadProjects:    true,
		SortEnvsByOrder: true,
	}
	cat, err := catalog.Load(loadOpts)
	assert.NoError(t, err)

	// Environments
	envs := cat.Environments
	assert.Equal(t, 2, len(envs))
	assert.Equal(t, "dev", envs[0].Name)
	assert.Equal(t, "production", envs[1].Name)

	// Projects
	projects := cat.Projects
	assert.Equal(t, 2, len(projects))
	assert.Equal(t, "project1", projects[0].Name)
	assert.Equal(t, "project2", projects[1].Name)

	// Cross-releases
	rels := cat.Releases.Items
	assert.Equal(t, 3, len(rels))
	AssertRelease(t, rels[0], "common-release", "1.2.4", "1.2.3")
	AssertRelease(t, rels[1], "dev-release", "2.2.2", "")
	AssertRelease(t, rels[2], "production-release", "", "1.1.1")
}

func AssertRelease(t *testing.T, crossRelease *cross.Release, name string, devVersion string, prodVersion string) {
	assert.Equal(t, name, crossRelease.Name)
	devRelease := crossRelease.Releases[0]
	prodRelease := crossRelease.Releases[1]

	if devVersion == "" {
		assert.Nil(t, devRelease)
	} else {
		assert.NotNil(t, devRelease)
		assert.Equal(t, devVersion, devRelease.Spec.Version)
	}

	if prodVersion == "" {
		assert.Nil(t, prodRelease)
	} else {
		assert.NotNil(t, prodRelease)
		assert.Equal(t, prodVersion, prodRelease.Spec.Version)
	}
}
