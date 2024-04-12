package main

import (
	"bytes"
	"cmp"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/pkg/catalog"
)

var (
	testCatalogURL  = cmp.Or(os.Getenv("CATALOG_TEST_REPO"), "git@github.com:nestoca/joy-catalog-test.git")
	testCatalogPath = filepath.Join(os.TempDir(), "catalog-test")
	testCache       = filepath.Join(os.TempDir(), "cache")
)

func TestReleaseRender(t *testing.T) {
	require.NoError(t, os.RemoveAll(testCache))
	require.NoError(t, os.MkdirAll(testCache, 0o755))

	require.NoError(t, os.RemoveAll(testCatalogPath))

	clone := exec.Command("git", "clone", testCatalogURL, testCatalogPath)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr

	require.NoError(t, clone.Run())

	t.Run("render not found", func(t *testing.T) {
		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "file://./testdata/charts",
					Name:    "base",
				},
			},
			DefaultChartRef: "generic",
			ValueMapping: &config.ValueMapping{
				Mappings: map[string]any{
					"image.tag": "{{ .Release.Spec.Version }}",
				},
			},
			JoyCache:           testCache,
			GitHubOrganization: "nestoca",
		}

		ctx := config.ToContext(context.Background(), cfg)

		cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
		require.NoError(t, err)

		ctx = catalog.ToContext(ctx, cat)

		var buffer bytes.Buffer

		cmd := NewReleaseRenderCmd()
		cmd.SetOut(&buffer)
		cmd.SetErr(&buffer)
		cmd.SetArgs([]string{"--color=false", "--env=testing", "does-not-exist"})

		require.EqualError(t, cmd.ExecuteContext(ctx), "getting release: not found: does-not-exist")
	})

	t.Run("diff", func(t *testing.T) {
		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "file://./testdata/charts",
					Name:    "base",
				},
			},
			DefaultChartRef: "generic",
			ValueMapping: &config.ValueMapping{
				Mappings: map[string]any{
					"image.tag": "{{ .Release.Spec.Version }}",
				},
			},
			JoyCache:           testCache,
			GitHubOrganization: "nestoca",
		}

		ctx := config.ToContext(context.Background(), cfg)

		cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
		require.NoError(t, err)

		ctx = catalog.ToContext(ctx, cat)

		var buffer bytes.Buffer

		cmd := NewReleaseRenderCmd()
		cmd.SetOut(&buffer)
		cmd.SetErr(&buffer)
		cmd.SetArgs([]string{
			"--color=false",
			"--env=testing",
			"--git-ref=test-release-with-diff",
			"--diff-ref=master",
			"test-release",
		})

		err = cmd.ExecuteContext(ctx)
		require.NoError(t, err, buffer.String())

		var removals, additions []string
		for _, line := range strings.Split(buffer.String(), "\n") {
			if strings.HasPrefix(line, "-  ") {
				removals = append(removals, strings.TrimSpace(line[1:]))
			}
			if strings.HasPrefix(line, "+  ") {
				additions = append(additions, strings.TrimSpace(line[1:]))
			}
		}

		require.Equal(t, []string{`image: "fake/image:0.0.1"`}, removals)
		require.Equal(t, []string{`image: "fake/image:0.0.1-test-diff"`}, additions)
	})

	t.Run("diff of new release", func(t *testing.T) {
		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "file://./testdata/charts",
					Name:    "base",
				},
			},
			DefaultChartRef: "generic",
			ValueMapping: &config.ValueMapping{
				Mappings: map[string]any{
					"image.tag": "{{ .Release.Spec.Version }}",
				},
			},
			JoyCache:           testCache,
			GitHubOrganization: "nestoca",
		}

		ctx := config.ToContext(context.Background(), cfg)

		cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
		require.NoError(t, err)

		ctx = catalog.ToContext(ctx, cat)

		var buffer bytes.Buffer

		cmd := NewReleaseRenderCmd()
		cmd.SetOut(&buffer)
		cmd.SetErr(&buffer)
		cmd.SetArgs([]string{
			"--color=false",
			"--env=testing",
			"--git-ref=test-new-release-diff",
			"--diff-ref=master",
			"other-release",
		})

		err = cmd.ExecuteContext(ctx)
		require.NoError(t, err, buffer.String())

		var removals, additions []string
		for _, line := range strings.Split(buffer.String(), "\n") {
			if strings.HasPrefix(line, "-  ") {
				removals = append(removals, strings.TrimSpace(line[1:]))
			}
			if strings.HasPrefix(line, "+  ") {
				additions = append(additions, strings.TrimSpace(line[1:]))
			}
		}

		require.Len(t, removals, 0)
		require.Greater(t, len(additions), 0)
	})

	t.Run("diff of new release inversed refs", func(t *testing.T) {
		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "file://./testdata/charts",
					Name:    "base",
				},
			},
			DefaultChartRef: "generic",
			ValueMapping: &config.ValueMapping{
				Mappings: map[string]any{
					"image.tag": "{{ .Release.Spec.Version }}",
				},
			},
			JoyCache:           testCache,
			GitHubOrganization: "nestoca",
		}

		ctx := config.ToContext(context.Background(), cfg)

		cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
		require.NoError(t, err)

		ctx = catalog.ToContext(ctx, cat)

		var buffer bytes.Buffer

		cmd := NewReleaseRenderCmd()
		cmd.SetOut(&buffer)
		cmd.SetErr(&buffer)
		cmd.SetArgs([]string{
			"--color=false",
			"--env=testing",
			"--git-ref=master",
			"--diff-ref=test-new-release-diff",
			"other-release",
		})

		err = cmd.ExecuteContext(ctx)
		require.NoError(t, err, buffer.String())

		var removals, additions []string
		for _, line := range strings.Split(buffer.String(), "\n") {
			if strings.HasPrefix(line, "-  ") {
				removals = append(removals, strings.TrimSpace(line[1:]))
			}
			if strings.HasPrefix(line, "+  ") {
				additions = append(additions, strings.TrimSpace(line[1:]))
			}
		}

		require.Len(t, additions, 0)
		require.Greater(t, len(removals), 0)
	})

	t.Run("diff release not found", func(t *testing.T) {
		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "file://./testdata/charts",
					Name:    "base",
				},
			},
			DefaultChartRef: "generic",
			ValueMapping: &config.ValueMapping{
				Mappings: map[string]any{
					"image.tag": "{{ .Release.Spec.Version }}",
				},
			},
			JoyCache:           testCache,
			GitHubOrganization: "nestoca",
		}

		ctx := config.ToContext(context.Background(), cfg)

		cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
		require.NoError(t, err)

		ctx = catalog.ToContext(ctx, cat)

		var buffer bytes.Buffer

		cmd := NewReleaseRenderCmd()
		cmd.SetOut(&buffer)
		cmd.SetErr(&buffer)
		cmd.SetArgs([]string{
			"--color=false",
			"--env=testing",
			"--git-ref=master",
			"--diff-ref=test-new-release-diff",
			"does-not-exist",
		})

		err = cmd.ExecuteContext(ctx)
		require.EqualError(t, err, "getting release: not found: does-not-exist")
	})
}
