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
	Refs            map[string]Chart
	DefaultChartRef string
	Root            string
	Puller
}

type Chart struct {
	RepoURL string `yaml:"repoUrl"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func (chart Chart) ToURL() (*url.URL, error) {
	uri, err := url.Parse(chart.RepoURL)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" {
		uri.Scheme = "oci"
		uri, err = url.Parse(uri.String())
		if err != nil {
			return nil, err
		}
	}

	return uri.JoinPath(chart.Name), nil
}

type ChartFS struct {
	Chart
	xfs.FS
}

// func merge(charts ...Chart) Chart {

// 	var result Chart
// 	for _, chart := range charts {
// 		result.Repo =
// 	}
// }

func (cache ChartCache) GetReleaseChart(release *v1alpha1.Release) (chart Chart, err error) {
	if repoURL := release.Spec.Chart.RepoUrl; repoURL != "" {
		return Chart{
			RepoURL: repoURL,
			Name:    release.Spec.Chart.Name,
			Version: release.Spec.Chart.Version,
		}, nil
	}

	ref := cmp.Or(release.Spec.Chart.Ref, cache.DefaultChartRef)

	chart = cache.Refs[ref]

	chart.Version = cmp.Or(
		release.Spec.Chart.Version,
		release.Environment.Spec.ChartVersions[ref],
		chart.Version,
	)

	return
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
