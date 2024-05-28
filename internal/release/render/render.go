package render

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"

	"github.com/Masterminds/sprig/v3"
	"github.com/davidmdm/x/xerr"
	"github.com/nestoca/survey/v2"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/pkg/helm"
)

type RenderParams struct {
	Release      *v1alpha1.Release
	Chart        *helm.ChartFS
	ValueMapping *config.ValueMapping
	Helm         helm.PullRenderer
}

func Render(ctx context.Context, params RenderParams) (string, error) {
	values, err := HydrateValues(params.Release, params.Chart, params.ValueMapping)
	if err != nil {
		return "", fmt.Errorf("hydrating values: %w", err)
	}

	opts := helm.RenderOpts{
		ReleaseName: params.Release.Name,
		ChartPath:   params.Chart.DirName(),
		Values:      values,
	}

	return params.Helm.Render(ctx, opts)
}

func getEnvironment(environments []*v1alpha1.Environment, name string) (*v1alpha1.Environment, error) {
	if name == "" {
		return environment.SelectSingle(environments, nil, "Select environment")
	}

	selectedEnv := environment.FindByName(environments, name)
	if selectedEnv == nil {
		return nil, NotFoundError(fmt.Sprintf("not found: %s", name))
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
		return nil, NotFoundError(fmt.Sprintf("not found within environment %s: %s", env, name))
	}

	return nil, NotFoundError(fmt.Sprintf("not found: %s", name))
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

func HydrateValues(release *v1alpha1.Release, chart *helm.ChartFS, mappings *config.ValueMapping) (map[string]any, error) {
	params := struct {
		Release     *v1alpha1.Release
		Environment *v1alpha1.Environment
	}{
		release,
		release.Environment,
	}

	// The following call has the side effect of making a deep copy of the values, which is necessary
	// for subsequent step to mutate the copy without affecting the original values.
	values, err := hydrateObjectValues(release.Spec.Values, params.Environment.Spec.Values)
	if err != nil {
		return nil, fmt.Errorf("hydrating object values: %w", err)
	}

	if mappings != nil && !slices.Contains(mappings.ReleaseIgnoreList, release.Name) {
		for mapping, value := range mappings.Mappings {
			setInMap(values, splitIntoPathSegments(mapping), value)
		}
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(string(data))
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

	result, err = unifyValues(result, chart)
	if err != nil {
		return nil, fmt.Errorf("unifying with chart schema: %w", err)
	}

	return result, nil
}

func unifyValues(values map[string]any, chart *helm.ChartFS) (map[string]any, error) {
	if chart == nil {
		return values, nil
	}

	rawSchema, err := chart.ReadFile("values.cue")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return values, nil
		}
		return nil, fmt.Errorf("reading values.cue: %w", err)
	}

	schema := cuecontext.New().
		CompileBytes(rawSchema).
		LookupPath(cue.MakePath(cue.Def("#values")))

	value := schema.Context().Encode(values)

	unified := schema.Unify(value)

	if err := unified.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		return nil, xerr.MultiErrFrom("validating values", AsErrorList(cueerrors.Errors(err))...)
	}

	var result map[string]any
	if err := unified.Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding values: %w", err)
	}

	return result, nil
}

var objectValuesRegex = regexp.MustCompile(`^\s*\$(\w+)\(\s*((\.\w+)+)\s*\)\s*$`)

const objectValuesSupportedPrefix = ".Environment.Spec.Values."

func hydrateObjectValues(values map[string]any, envValues map[string]any) (map[string]any, error) {
	resolvedValue, err := hydrateObjectValue(values, envValues)
	if err != nil {
		return nil, err
	}
	return resolvedValue.(map[string]any), err
}

func hydrateObjectValue(value any, envValues map[string]any) (any, error) {
	switch val := value.(type) {
	case string:
		operator, resolvedValue, err := resolveOperatorAndValue(val, envValues)
		if err != nil {
			return nil, err
		}
		if operator != "" && operator != "ref" {
			return nil, fmt.Errorf("only $ref() operator supported within object: %s", val)
		}
		return resolvedValue, nil
	case map[string]any:
		result := map[string]any{}
		for key, subValue := range val {
			resolvedValue, err := hydrateObjectValue(subValue, envValues)
			if err != nil {
				return nil, err
			}
			result[key] = resolvedValue
		}
		return result, nil
	case map[any]any:
		result := map[string]any{}
		for key, subValue := range val {
			resolvedValue, err := hydrateObjectValue(subValue, envValues)
			if err != nil {
				return nil, err
			}
			result[fmt.Sprint(key)] = resolvedValue
		}
		return result, nil
	case []any:
		var values []any
		for _, subValue := range val {
			switch subVal := subValue.(type) {
			case string:
				operator, resolvedValue, err := resolveOperatorAndValue(subVal, envValues)
				if err != nil {
					return nil, err
				}
				if operator == "spread" {
					resolvedSlice, ok := resolvedValue.([]any)
					if !ok {
						return nil, fmt.Errorf("$spread() operator must resolve to an array, but got: %T", resolvedValue)
					}
					values = append(values, resolvedSlice...)
				} else {
					values = append(values, resolvedValue)
				}
			default:
				resolvedValue, err := hydrateObjectValue(subVal, envValues)
				if err != nil {
					return nil, err
				}
				values = append(values, resolvedValue)
			}
		}
		return values, nil
	default:
		return value, nil
	}
}

func resolveOperatorAndValue(value string, envValues map[string]any) (string, any, error) {
	matches := objectValuesRegex.FindStringSubmatch(value)
	if len(matches) == 0 {
		return "", value, nil
	}

	operator := matches[1]
	if operator != "spread" && operator != "ref" {
		return "", nil, fmt.Errorf("unsupported object interpolation operator %q in expression: %s", operator, value)
	}

	fullPath := matches[2]
	if !strings.HasPrefix(fullPath, objectValuesSupportedPrefix) {
		return "", nil, fmt.Errorf("only %q prefix is supported for object interpolation, but found: %s", objectValuesSupportedPrefix, fullPath)
	}
	valuesPath := strings.Split(strings.TrimPrefix(fullPath, objectValuesSupportedPrefix), ".")
	resolvedValue, err := resolveObjectValue(envValues, valuesPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolving object value for path %q: %w", fullPath, err)
	}
	return operator, resolvedValue, nil
}

func resolveObjectValue(values map[string]any, path []string) (any, error) {
	key := path[0]
	value, ok := values[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in values", key)
	}
	if len(path) == 1 {
		return value, nil
	}
	mapValue, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("value for key %q is not a map", key)
	}
	return resolveObjectValue(mapValue, path[1:])
}

// setInMap modifies the map by adding the value to the path defined by segments.
// If the path defined by segments already exists, even if it points to a falsy value, this function does nothing.
// It will not overwrite any existing key/value pairs.
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

type NotFoundError string

func (err NotFoundError) Error() string { return string(err) }

func (NotFoundError) Is(err error) bool {
	_, ok := err.(NotFoundError)
	return ok
}

func IsNotFoundError(err error) bool {
	var notfound NotFoundError
	return errors.Is(err, notfound)
}

func AsErrorList[T error](list []T) []error {
	result := make([]error, len(list))
	for i, err := range list {
		result[i] = err
	}
	return result
}
