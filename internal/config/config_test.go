package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/pkg/helm"
)

func TestMergeCatalogResourceIntoCatalogConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		catalogConfig    Catalog
		catalogResource  v1alpha1.Catalog
		expected         Catalog
	}{
		{
			name: "catalog resource adds charts and default",
			catalogConfig: Catalog{
				Charts: map[string]helm.Chart{
					"joy-only": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy-only",
						Version: "1.0.0",
					},
				},
				DefaultChartRef: "joy-only",
			},
			catalogResource: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Charts: v1alpha1.CatalogCharts{
						Default: "catalog-chart",
						Refs: map[string]helm.Chart{
							"catalog-chart": {
								RepoURL: "catalog.example.com",
								Name:    "charts/catalog",
								Version: "2.0.0",
							},
						},
					},
				},
			},
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"joy-only": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy-only",
						Version: "1.0.0",
					},
					"catalog-chart": {
						RepoURL: "catalog.example.com",
						Name:    "charts/catalog",
						Version: "2.0.0",
					},
				},
				DefaultChartRef: "catalog-chart",
			},
		},
		{
			name: "catalog resource overrides joy config for same chart ref",
			catalogConfig: Catalog{
				Charts: map[string]helm.Chart{
					"shared": {
						RepoURL: "joy.example.com",
						Name:    "charts/from-joy",
						Version: "1.0.0",
					},
				},
				DefaultChartRef: "shared",
			},
			catalogResource: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Charts: v1alpha1.CatalogCharts{
						Default: "shared",
						Refs: map[string]helm.Chart{
							"shared": {
								RepoURL: "catalog.example.com",
								Name:    "charts/from-catalog",
								Version: "2.0.0",
								Mappings: map[string]any{
									"key1": "value1",
								},
							},
						},
					},
				},
			},
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"shared": {
						RepoURL: "catalog.example.com",
						Name:    "charts/from-catalog",
						Version: "2.0.0",
						Mappings: map[string]any{
							"key1": "value1",
						},
					},
				},
				DefaultChartRef: "shared",
			},
		},
		{
			name:            "empty catalog resource leaves joy config unchanged",
			catalogConfig: Catalog{
				Charts: map[string]helm.Chart{
					"joy-only": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy-only",
						Version: "1.0.0",
					},
				},
				DefaultChartRef: "joy-only",
			},
			catalogResource: v1alpha1.Catalog{},
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"joy-only": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy-only",
						Version: "1.0.0",
					},
				},
				DefaultChartRef: "joy-only",
			},
		},
		{
			name:          "nil charts map is initialized from catalog resource",
			catalogConfig: Catalog{},
			catalogResource: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Charts: v1alpha1.CatalogCharts{
						Default: "catalog-chart",
						Refs: map[string]helm.Chart{
							"catalog-chart": {
								RepoURL: "catalog.example.com",
								Name:    "charts/catalog",
								Version: "2.0.0",
							},
						},
					},
				},
			},
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"catalog-chart": {
						RepoURL: "catalog.example.com",
						Name:    "charts/catalog",
						Version: "2.0.0",
					},
				},
				DefaultChartRef: "catalog-chart",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			catalogConfig := tc.catalogConfig
			require.NoError(t, MergeCatalogResourceIntoCatalogConfig(&catalogConfig, &tc.catalogResource))
			require.Equal(t, tc.expected, catalogConfig)
		})
	}
}

func TestLoadCatalogConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		joyYAML  string
		catalogYAML string
		expected Catalog
	}{
		{
			name: "joy.yaml only",
			joyYAML: `
charts:
  joy-chart:
    repoUrl: joy.example.com
    name: charts/joy
    version: 1.0.0
defaultChartRef: joy-chart
`,
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"joy-chart": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy",
						Version: "1.0.0",
					},
				},
				DefaultChartRef: "joy-chart",
			},
		},
		{
			name: "catalog.yaml only",
			catalogYAML: `
apiVersion: joy.nesto.ca/v1alpha1
kind: Catalog
metadata:
  name: catalog
spec:
  charts:
    default: example-chart
    refs:
      example-chart:
        repoUrl: example.com
        name: charts/generic
        version: 1.2.3
        mappings:
          key1: value1
`,
			expected: Catalog{
				Charts: map[string]helm.Chart{
					"example-chart": {
						RepoURL: "example.com",
						Name:    "charts/generic",
						Version: "1.2.3",
						Mappings: map[string]any{
							"key1": "value1",
						},
					},
				},
				DefaultChartRef: "example-chart",
			},
		},
		{
			name: "catalog.yaml takes priority over joy.yaml",
			joyYAML: `
charts:
  shared:
    repoUrl: joy.example.com
    name: charts/from-joy
    version: 1.0.0
  joy-only:
    repoUrl: joy.example.com
    name: charts/joy-only
    version: 1.0.0
defaultChartRef: joy-only
minVersion: v1.0.0
`,
			catalogYAML: `
apiVersion: joy.nesto.ca/v1alpha1
kind: Catalog
metadata:
  name: catalog
spec:
  charts:
    default: example-chart
    refs:
      example-chart:
        repoUrl: example.com
        name: charts/generic
        version: 1.2.3
      shared:
        repoUrl: catalog.example.com
        name: charts/from-catalog
        version: 2.0.0
`,
			expected: Catalog{
				MinVersion: "v1.0.0",
				Charts: map[string]helm.Chart{
					"joy-only": {
						RepoURL: "joy.example.com",
						Name:    "charts/joy-only",
						Version: "1.0.0",
					},
					"example-chart": {
						RepoURL: "example.com",
						Name:    "charts/generic",
						Version: "1.2.3",
					},
					"shared": {
						RepoURL: "catalog.example.com",
						Name:    "charts/from-catalog",
						Version: "2.0.0",
					},
				},
				DefaultChartRef: "example-chart",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			catalogDir := t.TempDir()
			if tc.joyYAML != "" {
				require.NoError(t, os.WriteFile(filepath.Join(catalogDir, CatalogConfigFile), []byte(tc.joyYAML), 0o644))
			}
			if tc.catalogYAML != "" {
				require.NoError(t, os.WriteFile(filepath.Join(catalogDir, CatalogResourceFile), []byte(tc.catalogYAML), 0o644))
			}

			cfg, err := Load(context.Background(), catalogDir, catalogDir)
			require.NoError(t, err)
			require.Equal(t, tc.expected, cfg.Catalog)
		})
	}
}

func TestLoadCatalogConfigErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid catalog.yaml", func(t *testing.T) {
		t.Parallel()

		catalogDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(catalogDir, CatalogResourceFile), []byte(":\n\tbad"), 0o644))

		_, err := Load(context.Background(), catalogDir, catalogDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loading catalog resource")
	})

	t.Run("invalid joy.yaml", func(t *testing.T) {
		t.Parallel()

		catalogDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(catalogDir, CatalogConfigFile), []byte(":\n\tbad"), 0o644))

		_, err := Load(context.Background(), catalogDir, catalogDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loading catalog config")
	})
}
