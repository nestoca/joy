package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestReleasePromoteFilters(t *testing.T) {
	type TestFile struct {
		Path    string
		Content string
	}

	type TestCrossRelease struct {
		Name   string
		Source *TestFile
		Target *TestFile
	}

	cases := []struct {
		Name         string
		Args         []string
		Releases     []TestCrossRelease
		Err          string
		Expectations func(t *testing.T, files []*yml.File)
	}{
		{
			Name: "with explicit selection",
			Args: []string{"alpha"},
			Releases: []TestCrossRelease{
				{
					Name:   "alpha",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0}}"},
				},
				{
					Name:   "beta",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Expectations: func(t *testing.T, files []*yml.File) {
				require.Len(t, files, 1)

				require.Equal(t, "alpha-target.yaml", filepath.Base(files[0].Path))
				require.Equal(t, "{spec: {version: 1.2.3}}\n", string(files[0].Yaml))
			},
		},
		{
			Name: "with invalid explicit selection",
			Args: []string{"alpha", "delta"},
			Releases: []TestCrossRelease{
				{
					Name:   "alpha",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0}}"},
				},
				{
					Name:   "beta",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Err: "selecting releases to promote: release(s) not found: delta",
		},
		{
			Name: "all flag selects releases for update",
			Args: []string{"--all"},
			Releases: []TestCrossRelease{
				{
					Name:   "alpha",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0}}"},
				},
				{
					Name:   "beta",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Expectations: func(t *testing.T, files []*yml.File) {
				require.Len(t, files, 2)

				require.Equal(t, "alpha-target.yaml", filepath.Base(files[0].Path))
				require.Equal(t, "{spec: {version: 1.2.3}}\n", string(files[0].Yaml))

				require.Equal(t, "beta-target.yaml", filepath.Base(files[1].Path))
				require.Equal(t, "{spec: {version: 3.2.1}}\n", string(files[1].Yaml))
			},
		},
		{
			Name: "all flag with omit",
			Args: []string{"--all", "--omit=alpha"},
			Releases: []TestCrossRelease{
				{
					Name:   "alpha",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0}}"},
				},
				{
					Name:   "beta",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Expectations: func(t *testing.T, files []*yml.File) {
				require.Len(t, files, 1)
				require.Equal(t, "beta-target.yaml", filepath.Base(files[0].Path))
				require.Equal(t, "{spec: {version: 3.2.1}}\n", string(files[0].Yaml))
			},
		},
		{
			Name: "all flag with invalid omit",
			Args: []string{"--all", "--omit=does-not-exist"},
			Releases: []TestCrossRelease{
				{
					Name:   "alpha",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0}}"},
				},
				{
					Name:   "beta",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Err: "omitting releases: release(s) not found: does-not-exist",
		},
		{
			Name: "all flag with keep pre release",
			Args: []string{"--all", "--keep-prerelease"},
			Releases: []TestCrossRelease{
				{
					Name:   "prerelease-target",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0-some-feature}}"},
				},
				{
					Name:   "nil-target",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: &TestFile{Path: "beta-target.yaml", Content: "{spec: {version: 3.0.0}}"},
				},
			},
			Expectations: func(t *testing.T, files []*yml.File) {
				require.Len(t, files, 1)

				require.Equal(t, "beta-target.yaml", filepath.Base(files[0].Path))
				require.Equal(t, "{spec: {version: 3.2.1}}\n", string(files[0].Yaml))
			},
		},
		{
			Name: "all flag with keep pre release and nil target",
			Args: []string{"--all", "--keep-prerelease"},
			Releases: []TestCrossRelease{
				{
					Name:   "prerelease-target",
					Source: &TestFile{Path: "alpha-source.yaml", Content: "{spec: {version: 1.2.3}}"},
					Target: &TestFile{Path: "alpha-target.yaml", Content: "{spec: {version: 1.0.0-some-feature}}"},
				},
				{
					Name:   "nil-target",
					Source: &TestFile{Path: "beta-source.yaml", Content: "{spec: {version: 3.2.1}}"},
					Target: nil,
				},
			},
			Expectations: func(t *testing.T, files []*yml.File) {
				require.Len(t, files, 1)

				dir, filename := filepath.Split(files[0].Path)
				_, targetDir := filepath.Split(filepath.Clean(dir))

				require.Equal(t, "target/beta-source.yaml", filepath.Join(targetDir, filename))
				require.Equal(t, "{spec: {version: 3.2.1}}\n", string(files[0].Yaml))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			makeFile := func(t *testing.T, name, content string) *yml.File {
				t.Helper()

				f, err := yml.NewFile(name, []byte(content))
				require.NoError(t, err)

				return f
			}

			v1alpha1Release := func(release TestCrossRelease, env *v1alpha1.Environment, file *TestFile) *v1alpha1.Release {
				if file == nil {
					return nil
				}
				return &v1alpha1.Release{
					ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: release.Name},
					Environment:     env,
					File:            makeFile(t, filepath.Join(env.Dir, file.Path), file.Content),
					Project:         &v1alpha1.Project{},
				}
			}

			// We are giving a valid local path so that the relative path functionality
			// continues to work for promotions to nil targets
			cwd, err := os.Getwd()
			require.NoError(t, err)

			source := &v1alpha1.Environment{
				EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "source"},
				Dir:                 filepath.Join(cwd, "source"),
			}
			target := &v1alpha1.Environment{
				EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "target"},
				Spec: v1alpha1.EnvironmentSpec{
					Promotion: v1alpha1.Promotion{
						FromEnvironments: []string{source.Name},
					},
				},
				Dir: filepath.Join(cwd, "target"),
			}

			envs := []*v1alpha1.Environment{source, target}

			cat := &catalog.Catalog{
				Environments: envs,
				Releases: cross.ReleaseList{
					Environments: envs,
				},
			}

			for _, release := range tc.Releases {
				crossRelease := &cross.Release{
					Name: release.Name,
					Releases: []*v1alpha1.Release{
						v1alpha1Release(release, source, release.Source),
						v1alpha1Release(release, target, release.Target),
					},
				}
				cat.Releases.Items = append(cat.Releases.Items, crossRelease)
				for _, rel := range crossRelease.Releases {
					if rel == nil {
						continue
					}
					require.NoError(t, yaml.Unmarshal(rel.File.Yaml, &rel))
				}
			}

			ctx := catalog.ToContext(context.Background(), cat)
			ctx = config.ToContext(ctx, &config.Config{})

			writer := new(yml.WriterMock)

			cmd := NewReleasePromoteCmd(PromoteParams{
				Links:         new(links.ProviderMock),
				Info:          new(info.ProviderMock),
				Git:           new(promote.GitProviderMock),
				PullRequest:   new(pr.PullRequestProviderMock),
				Prompt:        new(promote.PromptProviderMock),
				Writer:        writer,
				PreRunConfigs: make(PreRunConfigs),
			})

			var buf bytes.Buffer
			cmd.SetOutput(&buf)
			cmd.SetErr(&buf)

			args := []string{
				"--no-prompt",
				"--source=source",
				"--target=target",
			}

			cmd.SetArgs(append(args, tc.Args...))

			if tc.Err != "" {
				require.EqualError(t, cmd.ExecuteContext(ctx), tc.Err)
				return
			}

			require.NoError(t, cmd.ExecuteContext(ctx))

			var files []*yml.File
			for _, call := range writer.WriteFileCalls() {
				files = append(files, call.File)
			}

			tc.Expectations(t, files)
		})
	}
}
