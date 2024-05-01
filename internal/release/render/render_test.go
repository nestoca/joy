package render

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestRender(t *testing.T) {
	type RenderTestParams struct {
		Env            string
		Release        string
		DefaultChart   string
		CacheDir       string
		Catalog        *catalog.Catalog
		IO             internal.IO
		SetupHelmMock  func(*helm.PullRendererMock)
		HelmAssertions func(*testing.T, *helm.PullRendererMock)
		ValueMapping   *config.ValueMapping
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
		stdin  bytes.Buffer
	)

	cases := []struct {
		Name          string
		Params        RenderTestParams
		ExpectedError string
		ExpectedOut   string
	}{
		{
			Name: "env not found",
			Params: RenderTestParams{
				Env: "quality-assurance",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
				},
			},
			ExpectedError: "getting environment: not found: quality-assurance",
		},
		{
			Name: "release not found",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{},
				},
			},
			ExpectedError: "getting release: not found: app",
		},
		{
			Name: "release not found in env",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "prod"}}},
								},
							},
						},
					},
				},
			},
			ExpectedError: "getting release: not found within environment qa: app",
		},
		{
			Name: "pull fails",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										Spec: v1alpha1.ReleaseSpec{
											Chart: v1alpha1.ReleaseChart{
												Version: "v1",
												Name:    "name",
												RepoUrl: "url",
											},
										},
									},
								},
							},
						},
					},
				},
				CacheDir: "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mock *helm.PullRendererMock) {
					mock.PullFunc = func(contextMoqParam context.Context, pullOptions helm.PullOptions) error {
						return errors.New("some informative error")
					}
				},
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.PullCalls(), 1)
					require.Equal(
						t,
						helm.PullOptions{
							Chart: helm.Chart{
								RepoURL: "url",
								Name:    "name",
								Version: "v1",
							},
							OutputDir: "~/.cache/joy/does_not_exist/url/name/v1",
						},
						mock.PullCalls()[0].PullOptions,
					)
				},
			},
			ExpectedError: "getting release chart: pulling chart: some informative error",
		},
		{
			Name: "pull fails with default chart",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										Spec: v1alpha1.ReleaseSpec{
											Chart: v1alpha1.ReleaseChart{
												Version: "v666",
											},
										},
									},
								},
							},
						},
					},
				},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mock *helm.PullRendererMock) {
					mock.PullFunc = func(contextMoqParam context.Context, pullOptions helm.PullOptions) error {
						return errors.New("some informative error")
					}
				},
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.PullCalls(), 1)
					require.Equal(
						t,
						helm.PullOptions{
							Chart: helm.Chart{
								RepoURL: "default",
								Name:    "chart",
								Version: "v666",
							},
							OutputDir: "~/.cache/joy/does_not_exist/default/chart/v666",
						},
						mock.PullCalls()[0].PullOptions,
					)
				},
			},
			ExpectedError: "getting release chart: pulling chart: some informative error",
		},
		{
			Name: "fail to hydrate values",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: "app"},
										Spec: v1alpha1.ReleaseSpec{
											Version: "v1.2.3",
											Values: map[string]any{
												"env":     "{{ .Environment.Name `!}}",
												"version": "{{ .Release.Spec.Version }}",
											},
										},
									},
								},
							},
						},
					},
				},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
			},
			ExpectedError: "hydrating values: template: :1: unterminated raw quoted string",
		},
		{
			Name: "fail to render",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: "app"},
										Spec: v1alpha1.ReleaseSpec{
											Version: "v1.2.3",
											Values: map[string]any{
												"env":     "{{ .Environment.Name }}",
												"version": "{{ .Release.Spec.Version }}",
											},
										},
									},
								},
							},
						},
					},
				},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mock *helm.PullRendererMock) {
					mock.RenderFunc = func(ctx context.Context, opts helm.RenderOpts) error {
						return errors.New("bebop")
					}
				},
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.PullCalls(), 1)
					require.Equal(
						t,
						helm.PullOptions{
							Chart: helm.Chart{
								RepoURL: "default",
								Name:    "chart",
								Version: "v1",
							},
							OutputDir: "~/.cache/joy/does_not_exist/default/chart/v1",
						},
						mock.PullCalls()[0].PullOptions,
					)
					require.Len(t, mock.RenderCalls(), 1)
					require.Equal(
						t,
						helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/v1/chart",
							Values: map[string]any{
								"env":     "qa",
								"version": "v1.2.3",
							},
						},
						mock.RenderCalls()[0].Opts,
					)
				},
			},
			ExpectedError: "rendering chart: bebop",
		},
		{
			Name: "render with chart mappings",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: "app"},
										Spec: v1alpha1.ReleaseSpec{
											Version: "v1.2.3",
											Values: map[string]any{
												"env":     "{{ .Environment.Name }}",
												"version": "{{ .Release.Spec.Version }}",
											},
										},
									},
								},
							},
						},
					},
				},
				ValueMapping: &config.ValueMapping{Mappings: map[string]any{
					"image.tag":             "{{ .Release.Spec.Version }}",
					`annotations.nesto\.ca`: true,
				}},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.PullCalls(), 1)
					require.Equal(
						t,
						helm.PullOptions{
							Chart: helm.Chart{
								RepoURL: "default",
								Name:    "chart",
								Version: "v1",
							},
							OutputDir: "~/.cache/joy/does_not_exist/default/chart/v1",
						},
						mock.PullCalls()[0].PullOptions,
					)
					require.Len(t, mock.RenderCalls(), 1)
					require.Equal(
						t,
						helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/v1/chart",
							Values: map[string]any{
								"env":         "qa",
								"version":     "v1.2.3",
								"image":       map[string]any{"tag": "v1.2.3"},
								"annotations": map[string]any{"nesto.ca": true},
							},
						},
						mock.RenderCalls()[0].Opts,
					)
				},
			},
		},
		{
			Name: "render with ignored chart mappings",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: "app"},
										Spec: v1alpha1.ReleaseSpec{
											Version: "v1.2.3",
											Values: map[string]any{
												"env":     "{{ .Environment.Name }}",
												"version": "{{ .Release.Spec.Version }}",
											},
										},
									},
								},
							},
						},
					},
				},
				ValueMapping: &config.ValueMapping{
					ReleaseIgnoreList: []string{"app"},
					Mappings: map[string]any{
						"image.tag":             "{{ .Release.Spec.Version }}",
						`annotations.nesto\.ca`: true,
					},
				},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.PullCalls(), 1)
					require.Equal(
						t,
						helm.PullOptions{
							Chart: helm.Chart{
								RepoURL: "default",
								Name:    "chart",
								Version: "v1",
							},
							OutputDir: "~/.cache/joy/does_not_exist/default/chart/v1",
						},
						mock.PullCalls()[0].PullOptions,
					)
					require.Len(t, mock.RenderCalls(), 1)
					require.Equal(
						t,
						helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/v1/chart",
							Values: map[string]any{
								"env":     "qa",
								"version": "v1.2.3",
							},
						},
						mock.RenderCalls()[0].Opts,
					)
				},
			},
		},
		{
			Name: "render with environment-level values",
			Params: RenderTestParams{
				Env:     "qa",
				Release: "app",
				Catalog: &catalog.Catalog{
					Environments: []*v1alpha1.Environment{
						{
							EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"},
							Spec:                v1alpha1.EnvironmentSpec{Values: map[string]any{"corsOrigins": []any{"origin1.com", "origin2.com"}}},
						},
					},
					Releases: cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: "app"},
										Spec: v1alpha1.ReleaseSpec{
											Version: "1.2.3",
											Values: map[string]any{
												"env":         "{{ .Environment.Name }}",
												"version":     "{{ .Release.Spec.Version }}",
												"corsOrigins": "$ref(.Environment.Spec.Values.corsOrigins)",
											},
										},
									},
								},
							},
						},
					},
				},
				DefaultChart: "generic",
				CacheDir:     "~/.cache/joy/does_not_exist",
				HelmAssertions: func(t *testing.T, mock *helm.PullRendererMock) {
					require.Len(t, mock.RenderCalls(), 1)
					require.Equal(
						t,
						helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/v1/chart",
							Values: map[string]any{
								"env":         "qa",
								"version":     "1.2.3",
								"corsOrigins": []any{"origin1.com", "origin2.com"},
							},
						},
						mock.RenderCalls()[0].Opts,
					)
				},
			},
		},
	}

	io := internal.IO{
		Out: &stdout,
		Err: &stderr,
		In:  &stdin,
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			stdout.Reset()

			helmMock := new(helm.PullRendererMock)

			if setup := tc.Params.SetupHelmMock; setup != nil {
				setup(helmMock)
			}
			if assertions := tc.Params.HelmAssertions; assertions != nil {
				defer assertions(t, helmMock)
			}

			// Use first environment as default for releases, because in some cases we need to have the
			// exact same environment in the release as in the catalog.
			if len(tc.Params.Catalog.Releases.Items) > 0 &&
				len(tc.Params.Catalog.Releases.Items[0].Releases) > 0 &&
				tc.Params.Catalog.Releases.Items[0].Releases[0].Environment == nil &&
				len(tc.Params.Catalog.Environments) > 0 {
				tc.Params.Catalog.Releases.Items[0].Releases[0].Environment = tc.Params.Catalog.Environments[0]
			}

			err := Render(context.Background(), RenderParams{
				Env:     tc.Params.Env,
				Release: tc.Params.Release,
				Cache: helm.ChartCache{
					Refs: map[string]helm.Chart{
						"generic": {
							RepoURL: "default",
							Name:    "chart",
							Version: "v1",
						},
					},
					DefaultChartRef: tc.Params.DefaultChart,
					Root:            tc.Params.CacheDir,
					Puller:          helmMock,
				},
				Catalog:            tc.Params.Catalog,
				CommonRenderParams: CommonRenderParams{ValueMapping: tc.Params.ValueMapping, IO: io, Helm: helmMock, Color: false},
			})
			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			}
			require.NoError(t, err)

			if tc.ExpectedOut != "" {
				require.Equal(t, tc.ExpectedOut, stdout.String())
			}
		})
	}
}

func TestSplitIntoPathSegments(t *testing.T) {
	cases := []struct {
		Input    string
		Segments []string
	}{
		{
			Input:    "common",
			Segments: []string{"common"},
		},
		{
			Input:    "left.right",
			Segments: []string{"left", "right"},
		},
		{
			Input:    ".",
			Segments: []string{"", ""},
		},
		{
			Input:    `\.`,
			Segments: []string{"."},
		},
		{
			Input:    `left.mid\.dle.right`,
			Segments: []string{"left", "mid.dle", "right"},
		},
		{
			Input:    `hello\\world`,
			Segments: []string{`hello\world`},
		},
		{
			Input:    `hello\\\.world`,
			Segments: []string{`hello\.world`},
		},
		{
			Input:    `hello\\.world`,
			Segments: []string{`hello\`, `world`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			require.Equal(t, tc.Segments, splitIntoPathSegments(tc.Input))
		})
	}
}

func TestSetInMap(t *testing.T) {
	cases := []struct {
		Name     string
		Segments []string
		Value    any
		Input    map[string]any
		Expected map[string]any
	}{
		{
			Name:     "top level",
			Segments: []string{"hello"},
			Value:    "world",
			Input:    map[string]any{},
			Expected: map[string]any{"hello": "world"},
		},
		{
			Name:     "top level value exists",
			Segments: []string{"hello"},
			Value:    "world",
			Input:    map[string]any{"hello": "bob"},
			Expected: map[string]any{"hello": "bob"},
		},
		{
			Name:     "creates nested objects",
			Segments: []string{"yes", "no"},
			Value:    "toaster",
			Input:    map[string]any{"hello": "world"},
			Expected: map[string]any{"hello": "world", "yes": map[string]any{"no": "toaster"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			setInMap(tc.Input, tc.Segments, tc.Value)
			require.Equal(t, tc.Expected, tc.Input)
		})
	}
}

func TestHydrateObjectValues(t *testing.T) {
	cases := []struct {
		Name           string
		EnvValues      map[string]any
		ReleaseValues  map[string]any
		ExpectedValues map[string]any
		ExpectedError  string
	}{
		{
			Name:      "top-level value",
			EnvValues: map[string]any{"corsOrigins": []any{"origin1.com", "origin2.com"}},
			ReleaseValues: map[string]any{
				"before":      "before",
				"corsOrigins": "$ref(.Environment.Spec.Values.corsOrigins)",
				"after":       "after",
			},
			ExpectedValues: map[string]any{
				"before":      "before",
				"corsOrigins": []any{"origin1.com", "origin2.com"},
				"after":       "after",
			},
		},
		{
			Name: "nested env value",
			EnvValues: map[string]any{
				"infra": map[string]any{
					"corsOrigins": []any{"origin1.com", "origin2.com"},
				},
			},
			ReleaseValues: map[string]any{
				"before":      "before",
				"corsOrigins": "$ref(.Environment.Spec.Values.infra.corsOrigins)",
				"after":       "after",
			},
			ExpectedValues: map[string]any{
				"before":      "before",
				"corsOrigins": []any{"origin1.com", "origin2.com"},
				"after":       "after",
			},
		},
		{
			Name: "nested release value",
			EnvValues: map[string]any{
				"corsOrigins": []any{"origin1.com", "origin2.com"},
			},
			ReleaseValues: map[string]any{
				"infra": map[string]any{
					"before":      "before",
					"corsOrigins": "$ref(.Environment.Spec.Values.corsOrigins)",
					"after":       "after",
				},
			},
			ExpectedValues: map[string]any{
				"infra": map[string]any{
					"before":      "before",
					"corsOrigins": []any{"origin1.com", "origin2.com"},
					"after":       "after",
				},
			},
		},
		{
			Name: "array ref within array",
			EnvValues: map[string]any{
				"corsOrigins": []any{"origin1.com", "origin2.com"},
			},
			ReleaseValues: map[string]any{
				"infra": []any{
					"before",
					"$ref(.Environment.Spec.Values.corsOrigins)",
					"after",
				},
			},
			ExpectedValues: map[string]any{
				"infra": []any{
					"before",
					[]any{"origin1.com", "origin2.com"},
					"after",
				},
			},
		},
		{
			Name: "ref within array within array",
			EnvValues: map[string]any{
				"corsOrigins": []any{"origin1.com", "origin2.com"},
			},
			ReleaseValues: map[string]any{
				"infra": []any{
					[]any{
						"before",
						"$ref(.Environment.Spec.Values.corsOrigins)",
						"after",
					},
				},
			},
			ExpectedValues: map[string]any{
				"infra": []any{
					[]any{
						"before",
						[]any{"origin1.com", "origin2.com"},
						"after",
					},
				},
			},
		},
		{
			Name: "ref within map within array",
			EnvValues: map[string]any{
				"corsOrigins": []any{"origin1.com", "origin2.com"},
			},
			ReleaseValues: map[string]any{
				"infra": []any{
					"before",
					map[string]any{
						"before": "before",
						"ref":    "$ref(.Environment.Spec.Values.corsOrigins)",
						"after":  "after",
					},
					"after",
				},
			},
			ExpectedValues: map[string]any{
				"infra": []any{
					"before",
					map[string]any{
						"before": "before",
						"ref":    []any{"origin1.com", "origin2.com"},
						"after":  "after",
					},
					"after",
				},
			},
		},
		{
			Name: "array spread within array",
			EnvValues: map[string]any{
				"corsOrigins": []any{"origin1.com", "origin2.com"},
			},
			ReleaseValues: map[string]any{
				"infra": []any{
					"before",
					"$spread(.Environment.Spec.Values.corsOrigins)",
					"after",
				},
			},
			ExpectedValues: map[string]any{
				"infra": []any{
					"before",
					"origin1.com",
					"origin2.com",
					"after",
				},
			},
		},
		{
			Name:      "preserve regular values",
			EnvValues: map[string]any{"corsOrigins": []any{"origin1.com", "origin2.com"}},
			ReleaseValues: map[string]any{
				"string":  "string value",
				"int":     42,
				"object":  map[string]any{"key": "value", "nested": map[string]any{"key": "value"}},
				"array":   []any{"a", "b", "c"},
				"env":     "{{ .Environment.Name }}",
				"version": "{{ .Release.Spec.Version }}",
			},
			ExpectedValues: map[string]any{
				"string":  "string value",
				"int":     42,
				"object":  map[string]any{"key": "value", "nested": map[string]any{"key": "value"}},
				"array":   []any{"a", "b", "c"},
				"env":     "{{ .Environment.Name }}",
				"version": "{{ .Release.Spec.Version }}",
			},
		},
		{
			Name:          "invalid operator",
			ReleaseValues: map[string]any{"key": "$invalid(.Environment.Spec.Values.corsOrigins)"},
			ExpectedError: `unsupported object interpolation operator "invalid" in expression: $invalid(.Environment.Spec.Values.corsOrigins)`,
		},
		{
			Name:          "invalid prefix",
			ReleaseValues: map[string]any{"key": "$ref(.Environment.Metadata.Name)"},
			ExpectedError: `only ".Environment.Spec.Values." prefix is supported for object interpolation, but found: .Environment.Metadata.Name`,
		},
		{
			Name:          "unsupported spread within object",
			EnvValues:     map[string]any{"corsOrigins": []any{"origin1.com", "origin2.com"}},
			ReleaseValues: map[string]any{"key": "$spread(.Environment.Spec.Values.corsOrigins)"},
			ExpectedError: `only $ref() operator supported within object: $spread(.Environment.Spec.Values.corsOrigins)`,
		},
		{
			Name:          "invalid path",
			EnvValues:     map[string]any{},
			ReleaseValues: map[string]any{"key": "$ref(.Environment.Spec.Values.does.not.exist)"},
			ExpectedError: `resolving object value for path ".Environment.Spec.Values.does.not.exist": key "does" not found in values`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			actualValues, err := hydrateObjectValues(tc.ReleaseValues, tc.EnvValues)

			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.ExpectedValues, actualValues)
		})
	}
}
