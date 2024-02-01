package render

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/pkg/catalog"
	"gopkg.in/yaml.v3"
)

type RenderOpts struct {
	Env          string
	Release      string
	DefaultChart string
	CacheDir     string
	Catalog      *catalog.Catalog
	IO           internal.IO
	Helm         helm.PullRenderer
	Color        bool
}

func Render(ctx context.Context, params RenderOpts) error {
	environment, err := getEnvironment(params.Catalog.Environments, params.Env)
	if err != nil {
		return fmt.Errorf("getting environment: %w", err)
	}

	release, err := getRelease(params.Catalog.Releases.Items, params.Release, environment.Name)
	if err != nil {
		return fmt.Errorf("getting release: %w", err)
	}

	chartURL, err := url.JoinPath(release.Spec.Chart.RepoUrl, release.Spec.Chart.Name)
	if err != nil {
		return fmt.Errorf("building chart url: %w", err)
	}
	if chartURL == "" {
		chartURL = params.DefaultChart
	}

	cachePath := filepath.Join(params.CacheDir, chartURL, release.Spec.Chart.Version)
	chartPath := filepath.Join(cachePath, filepath.Base(chartURL))

	if _, err := os.Stat(chartPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading cache: %w", err)
		}
		opts := helm.PullOptions{
			ChartURL:  chartURL,
			Version:   release.Spec.Chart.Version,
			OutputDir: cachePath,
		}
		if err := params.Helm.Pull(ctx, opts); err != nil {
			return fmt.Errorf("pulling helm chart: %w", err)
		}
	}

	values, err := hydrateValues(release, environment)
	if err != nil {
		fmt.Fprintln(params.IO.Out, "error hydrating values:", err)
		fmt.Fprintln(params.IO.Out, "fallback to raw release.spec.values")
		values = release.Spec.Values
	}

	dst := params.IO.Out
	if params.Color {
		dst = ManifestColorWriter{dst}
	}

	if err := params.Helm.Render(ctx, dst, chartPath, values); err != nil {
		return fmt.Errorf("rendering chart: %w", err)
	}

	return nil
}

func getEnvironment(environments []*v1alpha1.Environment, name string) (*v1alpha1.Environment, error) {
	if name == "" {
		return environment.SelectSingle(environments, nil, "Select environment")
	}

	selectedEnv := environment.FindByName(environments, name)
	if selectedEnv == nil {
		return nil, fmt.Errorf("not found: %s", name)
	}

	return selectedEnv, nil
}

func getRelease(releases []*cross.Release, name, env string) (*v1alpha1.Release, error) {
	if name == "" {
		return getReleaseViaPrompt(releases, env)
	}

	for _, crossRelease := range releases {
		if crossRelease.Name != name {
			continue
		}
		for _, release := range crossRelease.Releases {
			if release == nil {
				continue
			}
			if release.Environment.Name == env {
				return release, nil
			}
		}
		return nil, fmt.Errorf("not found within environment %s: %s", env, name)
	}

	return nil, fmt.Errorf("not found: %s", name)
}

func getReleaseViaPrompt(releases []*cross.Release, env string) (*v1alpha1.Release, error) {
	var (
		candidateNames    []string
		candidateReleases []*v1alpha1.Release
	)

	for _, crossRelease := range releases {
		for _, release := range crossRelease.Releases {
			if release == nil {
				continue
			}
			if release.Environment.Name == env {
				candidateNames = append(candidateNames, release.Name)
				candidateReleases = append(candidateReleases, release)
				break
			}
		}
	}

	var idx int
	if err := survey.AskOne(&survey.Select{Message: "Select release", Options: candidateNames, PageSize: 20}, &idx); err != nil {
		return nil, fmt.Errorf("failed prompt: %w", err)
	}

	return candidateReleases[idx], nil
}

func hydrateValues(release *v1alpha1.Release, environment *v1alpha1.Environment) (map[string]any, error) {
	values := release.Spec.Values

	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("").Parse(string(data))
	if err != nil {
		return nil, err
	}

	var builder bytes.Buffer
	params := struct {
		Release     *v1alpha1.Release
		Environment *v1alpha1.Environment
	}{
		release,
		environment,
	}

	if err := tmpl.Execute(&builder, params); err != nil {
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal(builder.Bytes(), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ManifestColorWriter colorizes helm manifest by searching for document breaks
// and source comments. The implementation is naive and depends on the write buffer
// not breaking lines. In theory this means colorization can fail, however in practice
// it works well enough.
type ManifestColorWriter struct {
	dst io.Writer
}

func (w ManifestColorWriter) Write(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "# Source:") {
			lines[i] = color.InYellow(line)
		}
	}

	n, err := w.dst.Write([]byte(strings.Join(lines, "\n")))
	return min(n, len(data)), err
}
