package main

import (
	"bytes"
	"context"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestBuildPromote(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		catalog       *catalog.Catalog
		expectedError string
	}{
		{
			name:    "allowed_pre_release",
			version: "2.3.4-rc1",
		},
		{
			name:          "disallowed_pre_release",
			version:       "2.3.4-rc1",
			catalog:       createCatalog(t, createParams{}),
			expectedError: "cannot promote pre-release version to staging environment",
		},
		{
			name:    "project_without_release",
			version: "2.3.4",
			catalog: createCatalog(t, createParams{
				getRelease: func(_ *v1alpha1.Release) *v1alpha1.Release {
					return nil
				},
			}),
			expectedError: "no releases found for project my-project",
		},
		{
			name:    "release_without_version",
			version: "2.3.4",
			catalog: createCatalog(t, createParams{
				getRelease: func(release *v1alpha1.Release) *v1alpha1.Release {
					release.Spec.Version = ""
					return release
				},
			}),
			expectedError: "release my-release has no version property: node not found for path 'spec.version': key 'version' does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			catalogDir := "."
			cat := tc.catalog
			if cat == nil {
				catalogDir = t.TempDir()
				err := cp.Copy("test_data/build_promote/"+tc.name+"/original", catalogDir)
				require.NoError(t, err)

				cat, err = catalog.Load(catalogDir, nil)
				require.NoError(t, err)
			}

			cfg := &config.Config{
				CatalogDir: catalogDir,
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

			diffs, err := testutils.CompareDirectories("test_data/build_promote/"+tc.name+"/expected", catalogDir)
			require.NoError(t, err)
			if len(diffs) > 0 {
				t.Errorf("unexpected differences:\n%s", diffs)
			}
		})
	}
}

type createParams struct {
	getEnvironment func(*v1alpha1.Environment) *v1alpha1.Environment
	getRelease     func(*v1alpha1.Release) *v1alpha1.Release
}

func createCatalog(t *testing.T, params createParams) *catalog.Catalog {
	var cat catalog.Catalog
	project := &v1alpha1.Project{
		ProjectMetadata: v1alpha1.ProjectMetadata{
			Name: "my-project",
		},
	}
	cat.Projects = []*v1alpha1.Project{project}

	env := &v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
			Name: "staging",
		},
	}
	if params.getEnvironment != nil {
		env = params.getEnvironment(env)
	}
	cat.Environments = []*v1alpha1.Environment{env}

	release := &v1alpha1.Release{
		ReleaseMetadata: v1alpha1.ReleaseMetadata{
			Name: "my-release",
		},
		Spec: v1alpha1.ReleaseSpec{
			Project: "my-project",
			Version: "1.2.3",
		},
	}
	if params.getRelease != nil {
		release = params.getRelease(release)
	}
	if release != nil {
		file, err := yml.NewFileFromObject("environments/staging/releases/my-release.yaml", 2, &release)
		require.NoError(t, err)
		release.File = file

		release.Project = project
		release.Environment = env

		cat.Releases = cross.ReleaseList{
			Environments: cat.Environments,
			Items: []*cross.Release{
				{
					Releases: []*v1alpha1.Release{
						release,
					},
				},
			},
		}
	}

	return &cat
}
