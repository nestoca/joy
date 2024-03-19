package catalog

import (
	"cmp"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/stretchr/testify/require"
)

var (
	testCatalogURL  = cmp.Or(os.Getenv("CATALOG_TEST_REPO"), "git@github.com:nestoca/joy-catalog-test.git")
	testCatalogPath = filepath.Join(os.TempDir(), "catalog-test")
)

func TestCatalogLoadE2E(t *testing.T) {
	require.NoError(t, os.RemoveAll(testCatalogPath))

	clone := exec.Command("git", "clone", testCatalogURL, testCatalogPath)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr

	require.NoError(t, clone.Run())

	cases := []struct {
		Name   string
		Branch string
		Error  string
	}{
		{
			Name:   "broken environment chart ref",
			Branch: "test-broken-chart-ref-environment",
			Error:  "validating environments: testing: validating chart references: unkown ref: missing-ref",
		},
		{
			Name:   "broken release chart ref",
			Branch: "test-broken-chart-ref-release",
			Error:  "validating releases: test-release/testing: invalid chart: unknown ref: missing-ref",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.NoError(t, git.Checkout(testCatalogPath, tc.Branch))

			cfg, err := config.LoadFile(filepath.Join(testCatalogPath, "joy.yaml"))
			require.NoError(t, err)

			_, err = Load(testCatalogPath, cfg.KnownChartRefs())
			require.EqualError(t, err, tc.Error)
		})
	}
}
