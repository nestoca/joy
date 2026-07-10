package helm

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/davidmdm/x/xfs"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/helm"
)

type (
	Chart   = helm.Chart
	ChartFS = helm.ChartFS
)

type ChartCache struct {
	Refs            map[string]Chart
	DefaultChartRef string
	Root            string
	Puller
}

func (cache ChartCache) GetReleaseChart(release *v1alpha1.Release) (Chart, error) {
	if repoURL := release.Spec.Chart.RepoUrl; repoURL != "" {
		return Chart{
			RepoURL:  repoURL,
			Name:     release.Spec.Chart.Name,
			Version:  release.Spec.Chart.Version,
			Mappings: release.Spec.Chart.Mappings,
		}, nil
	}

	ref := cmp.Or(release.Spec.Chart.Ref, cache.DefaultChartRef)

	chart := cache.Refs[ref]

	chart.Version = cmp.Or(
		release.Spec.Chart.Version,
		release.Environment.Spec.ChartVersions[ref],
		chart.Version,
	)

	chart.Mappings = func() map[string]any {
		mappings := make(map[string]any)
		maps.Copy(mappings, chart.Mappings)
		maps.Copy(mappings, release.Spec.Chart.Mappings)
		return mappings
	}()

	return chart, nil
}

func (cache ChartCache) GetReleaseChartFS(ctx context.Context, release *v1alpha1.Release) (*ChartFS, error) {
	chart, err := cache.GetReleaseChart(release)
	if err != nil {
		return nil, err
	}

	uri, err := chart.ToURL()
	if err != nil {
		return nil, fmt.Errorf("computing chart URL: %w", err)
	}

	if uri.Scheme == "file" {
		return &ChartFS{
			Chart: chart,
			FS:    xfs.Dir(filepath.Clean(strings.Join([]string{uri.Host, uri.Path}, "/"))),
		}, nil
	}

	versionDir := filepath.Join(cache.Root, uri.Host, uri.Path, chart.Version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return nil, err
	}

	chartDir := filepath.Join(versionDir, path.Base(uri.Path))

	if _, err := os.Stat(chartDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("verifying cache: %w", err)
		}

		pullOptions := PullOptions{Chart: chart, OutputDir: versionDir}
		if err := cache.Pull(ctx, pullOptions); err != nil {
			return nil, fmt.Errorf("pulling chart: %w", err)
		}
	}

	return &ChartFS{FS: xfs.Dir(chartDir), Chart: chart}, nil
}
