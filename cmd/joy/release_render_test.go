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
	t.Run("diff", func(t *testing.T) {
		require.NoError(t, os.RemoveAll(testCatalogPath))

		clone := exec.Command("git", "clone", testCatalogURL, testCatalogPath)
		clone.Stdout = os.Stdout
		clone.Stderr = os.Stderr

		require.NoError(t, clone.Run())

		cfg := &config.Config{
			CatalogDir: testCatalogPath,
			Charts: map[string]helm.Chart{
				"generic": {
					RepoURL: "northamerica-northeast1-docker.pkg.dev",
					Name:    "nesto-ci-78a3f2e6/charts/generic",
					Version: "1.16.0",
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
		cmd.SetOutput(&buffer)
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

		require.Equal(
			t,
			[]string{
				"tags.datadoghq.com/version: 0.0.1",
				"tags.datadoghq.com/version: 0.0.1",
				`image: "gcr.io/nesto-ci-78a3f2e6/test-release/api:0.0.1"`,
				`value: "0.0.1"`,
			},
			removals,
		)

		require.Equal(
			t,
			[]string{
				"tags.datadoghq.com/version: 0.0.1-test-diff",
				"tags.datadoghq.com/version: 0.0.1-test-diff",
				`image: "gcr.io/nesto-ci-78a3f2e6/test-release/api:0.0.1-test-diff"`,
				`value: "0.0.1-test-diff"`,
			},
			additions,
		)
	})
}
