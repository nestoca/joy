package render

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/davidmdm/x/xfs"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/pkg/helm"
)

func TestRenderRelease(t *testing.T) {
	type RenderTestParams struct {
		Release       *v1alpha1.Release
		Chart         helm.Chart
		ChartFS       func(*xfs.FSMock) func(*testing.T)
		IO            internal.IO
		SetupHelmMock func(*helm.PullRendererMock) func(*testing.T)
	}

	var stdout bytes.Buffer

	buildRelease := func(env, name string) *v1alpha1.Release {
		return &v1alpha1.Release{
			ReleaseMetadata: v1alpha1.ReleaseMetadata{Name: name},
			Environment:     &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: env}},
		}
	}

	cases := []struct {
		Name          string
		Params        RenderTestParams
		ExpectedError string
		ExpectedOut   string
	}{
		{
			Name: "fail to hydrate values",
			Params: RenderTestParams{
				Release: buildRelease("env", "release"),
				Chart: helm.Chart{
					Mappings: map[string]any{
						"image.tag": "{{ .Release.Spec.Version }",
					},
				},
				ChartFS: func(f *xfs.FSMock) func(*testing.T) {
					return func(t *testing.T) {}
				},
			},
			ExpectedError: `hydrating values: template: :2: unexpected "}" in operand`,
		},
		{
			Name: "fail to render",
			Params: RenderTestParams{
				Release: buildRelease("env", "release"),
				SetupHelmMock: func(mock *helm.PullRendererMock) func(*testing.T) {
					mock.RenderFunc = func(ctx context.Context, opts helm.RenderOpts) (string, error) {
						return "", errors.New("bebop")
					}

					return func(t *testing.T) {
						require.Len(t, mock.RenderCalls(), 1)
						require.Equal(
							t,
							helm.RenderOpts{
								ReleaseName: "release",
								Namespace:   "default",
								Values:      map[string]any{},
								ChartPath:   "path/to/chart",
							},
							mock.RenderCalls()[0].Opts,
						)
					}
				},
			},
			ExpectedError: "bebop",
		},
		{
			Name: "render with environment-level values",
			Params: RenderTestParams{
				Release: func() *v1alpha1.Release {
					rel := buildRelease("env", "release")
					rel.Spec.Values = map[string]any{
						"env":         "{{ .Environment.Name }}",
						"version":     "{{ .Release.Spec.Version }}",
						"corsOrigins": "$ref(.Environment.Spec.Values.corsOrigins)",
					}
					rel.Spec.Version = "v1.2.3"
					rel.Environment.Spec.Values = map[string]any{
						"corsOrigins": []any{"origin1.com", "origin2.com"},
					}
					return rel
				}(),
				SetupHelmMock: func(mock *helm.PullRendererMock) func(*testing.T) {
					return func(t *testing.T) {
						require.Len(t, mock.RenderCalls(), 1)
						require.Equal(
							t,
							helm.RenderOpts{
								ReleaseName: "release",
								Namespace:   "default",
								Values: map[string]any{
									"corsOrigins": []any{"origin1.com", "origin2.com"},
									"env":         "env",
									"version":     "v1.2.3",
								},
								ChartPath: "path/to/chart",
							},
							mock.RenderCalls()[0].Opts,
						)
					}
				},
			},
		},
		{
			Name: "with chart level mappings",
			Params: RenderTestParams{
				Release: func() *v1alpha1.Release {
					rel := buildRelease("env", "release")
					rel.Spec.Values = map[string]any{}
					rel.Spec.Version = "v9.9.9"
					return rel
				}(),
				Chart: helm.Chart{
					Mappings: map[string]any{
						"test.mapping": "{{ .Release.Spec.Version }}",
					},
				},
				SetupHelmMock: func(mock *helm.PullRendererMock) func(*testing.T) {
					return func(t *testing.T) {
						require.Len(t, mock.RenderCalls(), 1)
						require.Equal(
							t,
							helm.RenderOpts{
								ReleaseName: "release",
								Namespace:   "default",
								Values: map[string]any{
									"test": map[string]any{
										"mapping": "v9.9.9",
									},
								},
								ChartPath: "path/to/chart",
							},
							mock.RenderCalls()[0].Opts,
						)
					}
				},
			},
		},
		{
			Name: "chart mappings take priority over global mappings",
			Params: RenderTestParams{
				Release: func() *v1alpha1.Release {
					rel := buildRelease("env", "release")
					rel.Spec.Values = map[string]any{}
					rel.Spec.Version = "v9.9.9"
					return rel
				}(),
				Chart: helm.Chart{
					Mappings: map[string]any{"test.mapping": "{{ .Release.Spec.Version }}"},
				},
				SetupHelmMock: func(mock *helm.PullRendererMock) func(*testing.T) {
					return func(t *testing.T) {
						require.Len(t, mock.RenderCalls(), 1)
						require.Equal(
							t,
							helm.RenderOpts{
								ReleaseName: "release",
								Namespace:   "default",
								Values: map[string]any{
									"test": map[string]any{
										"mapping": "v9.9.9",
									},
								},
								ChartPath: "path/to/chart",
							},
							mock.RenderCalls()[0].Opts,
						)
					}
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			stdout.Reset()

			helmMock := new(helm.PullRendererMock)

			if setup := tc.Params.SetupHelmMock; setup != nil {
				defer setup(helmMock)(t)
			}

			fsMock := &xfs.FSMock{
				DirNameFunc: func() string {
					return "path/to/chart"
				},
				ReadFileFunc: func(name string) ([]byte, error) {
					return nil, os.ErrNotExist
				},
			}
			if setup := tc.Params.ChartFS; setup != nil {
				defer setup(fsMock)(t)
			}

			defer func() {
				if err := recover(); err != nil {
					require.NoError(t, fmt.Errorf("%v", err))
				}
			}()

			result, err := Render(context.Background(), RenderParams{
				Release: tc.Params.Release,
				Chart: &helm.ChartFS{
					Chart: tc.Params.Chart,
					FS:    fsMock,
				},
				Helm: helmMock,
			})

			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.ExpectedOut, result)
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

func TestSchemaUnification(t *testing.T) {
	cases := []struct {
		Name           string
		ReadFileFunc   func(string) ([]byte, error)
		Values         map[string]any
		ExpectedValues map[string]any
		ExpectedError  string
	}{
		{
			Name: "applies schema default",
			ReadFileFunc: func(name string) ([]byte, error) {
				return []byte(`#values: { color: "r" | "g" | *"b" }`), nil
			},
			Values:         map[string]any{},
			ExpectedValues: map[string]any{"color": "b"},
		},
		{
			Name: "fails schema validation",
			ReadFileFunc: func(name string) ([]byte, error) {
				return []byte(`#values: { color: "r" | "g" | *"b" }`), nil
			},
			Values: map[string]any{"color": "cyan", "enabled": true},
			ExpectedError: strings.Join(
				[]string{
					"unifying with chart schema: validating values:",
					"  - #values.color: 3 errors in empty disjunction:",
					`  - #values.color: conflicting values "b" and "cyan"`,
					`  - #values.color: conflicting values "g" and "cyan"`,
					`  - #values.color: conflicting values "r" and "cyan"`,
				},
				"\n",
			),
		},
		{
			Name: "no schema file",
			ReadFileFunc: func(name string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			Values:         map[string]any{"color": "red"},
			ExpectedValues: map[string]any{"color": "red"},
		},
		{
			Name: "do not html escape strings",
			ReadFileFunc: func(name string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			Values:         map[string]any{"successCondition": "x > 100"},
			ExpectedValues: map[string]any{"successCondition": "x > 100"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mockFS := &xfs.FSMock{ReadFileFunc: tc.ReadFileFunc}

			result, err := HydrateValues(
				&v1alpha1.Release{
					Spec:        v1alpha1.ReleaseSpec{Values: tc.Values},
					Environment: &v1alpha1.Environment{},
				},
				&helm.ChartFS{FS: mockFS},
			)

			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.ExpectedValues, result)
		})
	}
}
