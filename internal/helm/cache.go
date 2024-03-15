package helm

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/davidmdm/x/xfs"
	"github.com/nestoca/joy/api/v1alpha1"
)

type ChartCache struct {
	Refs         map[string]Chart
	DefaultRef   string
	DefaultChart string
	Root         string
	Puller
}

type ChartFS struct {
	Chart
	xfs.FS
}

func (cache ChartCache) GetReleaseChart(release *v1alpha1.Release) (Chart, error) {
	if legacyChart := release.Spec.Chart.LegacyReleaseChart; (legacyChart != v1alpha1.LegacyReleaseChart{}) {
		return parseChartURL(fmt.Sprintf("%s/%s:%s", legacyChart.RepoUrl, legacyChart.Name, release.Spec.Chart.Version))
	}

	if url := release.Spec.Chart.URL; url != "" {
		return Chart{
			URL:     url,
			Version: release.Spec.Chart.Version,
		}, nil
	}

	if ref := cmp.Or(release.Spec.Chart.Ref, cache.DefaultRef); ref != "" {
		chart := cache.Refs[ref]
		if version := release.Environment.Spec.ChartVersions[ref]; version != "" {
			chart.Version = version
		}
		return chart, nil
	}

	return parseChartURL(fmt.Sprintf("%s:%s", cache.DefaultChart, release.Spec.Chart.Version))
}

func (cache ChartCache) GetReleaseChartFS(ctx context.Context, release *v1alpha1.Release) (*ChartFS, error) {
	chart, err := cache.GetReleaseChart(release)
	if err != nil {
		return nil, fmt.Errorf("inferring chart from release: %w", err)
	}

	uri, err := url.Parse(chart.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing chart url: %w", err)
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
