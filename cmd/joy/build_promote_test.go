package main

import (
	"bytes"
	"context"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestBuildPromote(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		catalog       *catalog.Catalog
		chartVersion  string
		expectedError string
	}{
		{
			name:    "allowed_pre_release",
			version: "2.3.4-rc1",
		},
		{
			name:         "allowed_pre_release_with_chart",
			version:      "2.3.4-rc1",
			chartVersion: "5.6.7",
		},
		{
			name:          "disallowed_pre_release",
			version:       "2.3.4-rc1",
			catalog:       newBuildPromoteTestCatalog(t, newTestCatalogParams{}),
			expectedError: "cannot promote prerelease version to staging environment",
		},
		{
			name:    "project_without_release",
			version: "2.3.4",
			catalog: newBuildPromoteTestCatalog(t, newTestCatalogParams{
				noReleases: true,
			}),
			expectedError: "no releases found for project my-project",
		},
		{
			name:    "release_without_version",
			version: "2.3.4",
			catalog: newBuildPromoteTestCatalog(t, newTestCatalogParams{
				releaseFunc: func(release *v1alpha1.Release) {
					release.Spec.Version = ""
				},
			}),
			expectedError: "release my-project has no version property: node not found for path 'spec.version': key 'version' does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			catalogDir := "."
			cat := tc.catalog
			if cat == nil {
				catalogDir = t.TempDir()
				err := cp.Copy("testdata/build_promote/"+tc.name+"/original", catalogDir)
				require.NoError(t, err)

				cat, err = catalog.Load(catalogDir, nil)
				require.NoError(t, err)
			}

			cfg := &config.Config{
				User: config.User{CatalogDir: catalogDir},
			}
			ctx := config.ToContext(context.Background(), cfg)
			ctx = catalog.ToContext(ctx, cat)

			var buffer bytes.Buffer

			cmd := NewBuildPromoteCmd()
			cmd.SetOut(&buffer)
			cmd.SetErr(&buffer)
			cmd.SetArgs([]string{
				"staging",
				"my-project",
				tc.version,
			})
			if tc.chartVersion != "" {
				err := cmd.Flags().Set("chart-version", tc.chartVersion)
				require.NoError(t, err)
			}

			err := cmd.ExecuteContext(ctx)

			if tc.expectedError != "" {
				actualError := ""
				if err != nil {
					actualError = err.Error()
				}
				require.Equal(t, tc.expectedError, actualError)
				return
			}

			require.NoError(t, err, buffer.String())

			diffs, err := testutils.CompareDirectories("testdata/build_promote/"+tc.name+"/expected", catalogDir)
			require.NoError(t, err)
			if len(diffs) > 0 {
				t.Errorf("unexpected differences:\n%s", diffs)
			}
		})
	}
}

type newTestCatalogParams struct {
	projectFunc func(*v1alpha1.Project)
	releaseFunc func(*v1alpha1.Release)
	noReleases  bool
}

func newBuildPromoteTestCatalog(t *testing.T, params newTestCatalogParams) *catalog.Catalog {
	builder := catalog.NewBuilder(t)
	staging := builder.AddEnvironment("staging", nil)
	project := builder.AddProject("my-project", params.projectFunc)
	if !params.noReleases {
		builder.AddRelease(staging, project, "1.2.3", params.releaseFunc)
	}
	return builder.Build()
}
