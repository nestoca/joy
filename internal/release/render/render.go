package render

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/TwiN/go-color"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/pkg/catalog"
)

type RenderParams struct {
	Env     string
	Release string
	Cache   helm.ChartCache
	Catalog *catalog.Catalog
	CommonRenderParams
}

type CommonRenderParams struct {
	ValueMapping *config.ValueMapping
	IO           internal.IO
	Helm         helm.PullRenderer
	Color        bool
}

func Render(ctx context.Context, params RenderParams) error {
	environment, err := getEnvironment(params.Catalog.Environments, params.Env)
	if err != nil {
		return fmt.Errorf("getting environment: %w", err)
	}

	release, err := getRelease(params.Catalog.Releases.Items, params.Release, environment.Name)
	if err != nil {
		return fmt.Errorf("getting release: %w", err)
	}

	chart, err := params.Cache.GetReleaseChart(ctx, release)
	if err != nil {
		return fmt.Errorf("getting release chart: %w", err)
	}

	return RenderRelease(ctx, RenderReleaseParams{
		Release:            release,
		Chart:              chart,
		CommonRenderParams: params.CommonRenderParams,
	})
}

type RenderReleaseParams struct {
	Release *v1alpha1.Release
	Chart   *helm.Chart
	CommonRenderParams
}

func RenderRelease(ctx context.Context, params RenderReleaseParams) error {
	values, err := HydrateValues(params.Release, params.Release.Environment, params.ValueMapping)
	if err != nil {
		fmt.Fprintln(params.IO.Out, "error hydrating values:", err)
		fmt.Fprintln(params.IO.Out, "fallback to raw release.spec.values")
		values = params.Release.Spec.Values
	}

	dst := params.IO.Out
	if params.Color {
		dst = ManifestColorWriter{dst}
	}

	opts := helm.RenderOpts{
		Dst:         dst,
		ReleaseName: params.Release.Name,
		ChartPath:   params.Chart.DirName(),
		Values:      values,
	}

	if err := params.Helm.Render(ctx, opts); err != nil {
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

func HydrateValues(release *v1alpha1.Release, environment *v1alpha1.Environment, mappings *config.ValueMapping) (map[string]any, error) {
	params := struct {
		Release     *v1alpha1.Release
		Environment *v1alpha1.Environment
	}{
		release,
		environment,
	}

	values := maps.Clone(release.Spec.Values)
	if mappings != nil && !slices.Contains(mappings.ReleaseIgnoreList, release.Name) {
		for mapping, value := range mappings.Mappings {
			setInMap(values, splitIntoPathSegments(mapping), value)
		}
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("").Parse(string(data))
	if err != nil {
		return nil, err
	}

	var builder bytes.Buffer
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

// setInMap modifies the map by adding the value to the path defined by segments.
// If the path defined by segments already exists, even if it points to a falsy value, this function does nothing.
// It will not overwite any existing key/value pairs.
func setInMap(mapping map[string]any, segments []string, value any) {
	for i, key := range segments {
		if i == len(segments)-1 {
			if _, ok := mapping[key]; !ok {
				mapping[key] = value
			}
			return
		}

		subValue, ok := mapping[key]
		if !ok {
			submap := map[string]any{}
			mapping[key] = submap
			mapping = submap
			continue
		}

		submap, ok := subValue.(map[string]any)
		if !ok {
			return
		}
		mapping = submap
	}
}

func splitIntoPathSegments(input string) (result []string) {
	var (
		start   int
		escaped bool
	)

	sanitize := func(value string) string {
		value = strings.ReplaceAll(value, `\.`, ".")
		value = strings.ReplaceAll(value, `\\`, `\`)
		return value
	}

	for i, c := range input {
		switch c {
		case '\\':
			escaped = !escaped
		case '.':
			if escaped {
				continue
			}
			result = append(result, sanitize(input[start:i]))
			escaped = false
			start = i + 1
		default:
			escaped = false
		}
	}

	result = append(result, sanitize(input[start:]))

	return
}
