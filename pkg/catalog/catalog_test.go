package catalog

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/cross"
)

func TestCatalogLoadE2E(t *testing.T) {
	cases := []struct {
		Name   string
		Folder string
		Error  string
	}{
		{
			Name:   "broken environment chart ref",
			Folder: "broken-chart-ref-environment",
			Error:  "validating environments: testing: validating chart references: unknown ref: missing-ref",
		},
		{
			Name:   "broken release chart ref",
			Folder: "broken-chart-ref-release",
			Error:  "validating releases: test-release/testing: invalid chart: unknown ref: missing-ref",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var cfg config.Config
			err := config.LoadFile(filepath.Join("testdata", tc.Folder, "joy.yaml"), &cfg.Catalog)
			require.NoError(t, err)

			_, err = Load(filepath.Join("testdata", tc.Folder), cfg.KnownChartRefs())
			require.EqualError(t, err, tc.Error)
		})
	}
}

func TestFreeformEnvsAndReleasesLoading(t *testing.T) {
	catalogDir, err := filepath.Abs("testdata/freeform")
	require.NoError(t, err)

	cat, err := Load(catalogDir, nil)
	require.NoError(t, err)

	envs := cat.Environments
	require.Equal(t, 2, len(envs))
	require.Equal(t, "dev", envs[0].Name)
	require.Equal(t, "production", envs[1].Name)

	projects := cat.Projects
	require.Equal(t, 2, len(projects))
	require.Equal(t, "project1", projects[0].Name)
	require.Equal(t, "project2", projects[1].Name)

	rels := cat.Releases.Items
	require.Equal(t, 3, len(rels))
	requireRelease(t, rels[0], "common-release", "1.2.4", "1.2.3")
	requireRelease(t, rels[1], "dev-release", "2.2.2", "")
	requireRelease(t, rels[2], "production-release", "", "1.1.1")
}

func requireRelease(t *testing.T, crossRelease *cross.Release, name string, devVersion string, prodVersion string) {
	require.Equal(t, name, crossRelease.Name)
	devRelease := crossRelease.Releases[0]
	prodRelease := crossRelease.Releases[1]

	if devVersion == "" {
		require.Nil(t, devRelease)
	} else {
		require.NotNil(t, devRelease)
		require.Equal(t, devVersion, devRelease.Spec.Version)
	}

	if prodVersion == "" {
		require.Nil(t, prodRelease)
	} else {
		require.NotNil(t, prodRelease)
		require.Equal(t, prodVersion, prodRelease.Spec.Version)
	}
}
