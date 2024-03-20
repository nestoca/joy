package helm

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/davidmdm/x/xfs"

	"github.com/nestoca/joy/api/v1alpha1"
)

type ChartCache struct {
	DefaultChart string
	Root         string
	Puller
}

type Chart struct {
	URL     string
	Version string
	xfs.FS
}

func (charts ChartCache) GetReleaseChart(ctx context.Context, release *v1alpha1.Release) (*Chart, error) {
	chartURL, err := toURL(cmp.Or(release.Spec.Chart.RepoUrl, charts.DefaultChart), "oci")
	if err != nil {
		return nil, fmt.Errorf("parsing release.spec.chart.repoUrl: %w", err)
	}

	chartURL = chartURL.JoinPath(release.Spec.Chart.Name)

	versionDir := filepath.Join(charts.Root, chartURL.Host, chartURL.Path, release.Spec.Chart.Version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return nil, err
	}

	chartDir := filepath.Join(versionDir, filepath.Base(chartURL.Path))

	if _, err := os.Stat(chartDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("verifying cache: %w", err)
		}

		pullOptions := PullOptions{
			ChartURL:  chartURL.String(),
			Version:   release.Spec.Chart.Version,
			OutputDir: versionDir,
		}
		if err := charts.Pull(ctx, pullOptions); err != nil {
			return nil, fmt.Errorf("pulling chart: %w", err)
		}
	}

	return &Chart{
		URL:     chartURL.String(),
		Version: release.Spec.Chart.Version,
		FS:      xfs.Dir(chartDir),
	}, nil
}

func toURL(value, defaultScheme string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = defaultScheme
	}

	return url.Parse(u.String())
}
