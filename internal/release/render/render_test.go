package render

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/pkg/catalog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRender(t *testing.T) {
	type RenderTestParams struct {
		Env           string
		Release       string
		DefaultChart  string
		CacheDir      string
		Catalog       *catalog.Catalog
		IO            internal.IO
		SetupHelmMock func(*helm.MockPullRenderer)
		ChartMappings map[string]any
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
					Releases: &cross.ReleaseList{},
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
					Releases: &cross.ReleaseList{
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
					Releases: &cross.ReleaseList{
						Items: []*cross.Release{
							{
								Name: "app",
								Releases: []*v1alpha1.Release{
									{
										Spec: v1alpha1.ReleaseSpec{
											Chart: v1alpha1.ReleaseChart{
												Name:    "name",
												RepoUrl: "url",
												Version: "v1",
											},
										},
										Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
									},
								},
							},
						},
					},
				},
				CacheDir: "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mpr *helm.MockPullRenderer) {
					mpr.EXPECT().
						Pull(context.Background(), helm.PullOptions{
							ChartURL:  "url/name",
							Version:   "v1",
							OutputDir: "~/.cache/joy/does_not_exist/url/name/v1",
						}).
						Return(errors.New("some informative error"))
				},
			},
			ExpectedError: "pulling helm chart: some informative error",
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
					Releases: &cross.ReleaseList{
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
										Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
									},
								},
							},
						},
					},
				},
				DefaultChart: "default/chart",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mpr *helm.MockPullRenderer) {
					mpr.EXPECT().
						Pull(context.Background(), helm.PullOptions{
							ChartURL:  "default/chart",
							Version:   "v666",
							OutputDir: "~/.cache/joy/does_not_exist/default/chart/v666",
						}).
						Return(errors.New("some informative error"))
				},
			},
			ExpectedError: "pulling helm chart: some informative error",
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
					Releases: &cross.ReleaseList{
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
										Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
									},
								},
							},
						},
					},
				},
				DefaultChart: "default/chart",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mpr *helm.MockPullRenderer) {
					mpr.EXPECT().
						Pull(context.Background(), helm.PullOptions{
							ChartURL:  "default/chart",
							Version:   "",
							OutputDir: "~/.cache/joy/does_not_exist/default/chart",
						}).
						Return(nil)

					mpr.EXPECT().
						Render(context.Background(), helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/chart",
							Values: map[string]any{
								"env":     "{{ .Environment.Name `!}}",
								"version": "{{ .Release.Spec.Version }}",
							},
						}).
						Return(nil)
				},
			},
			ExpectedOut: "error hydrating values: template: :1: unterminated raw quoted string\nfallback to raw release.spec.values\n",
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
					Releases: &cross.ReleaseList{
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
										Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
									},
								},
							},
						},
					},
				},
				DefaultChart: "default/chart",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mpr *helm.MockPullRenderer) {
					mpr.EXPECT().
						Pull(context.Background(), helm.PullOptions{
							ChartURL:  "default/chart",
							Version:   "",
							OutputDir: "~/.cache/joy/does_not_exist/default/chart",
						}).
						Return(nil)

					mpr.EXPECT().
						Render(context.Background(), helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/chart",
							Values: map[string]any{
								"env":     "qa",
								"version": "v1.2.3",
							},
						}).
						Return(errors.New("bebop"))
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
					Releases: &cross.ReleaseList{
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
										Environment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "qa"}},
									},
								},
							},
						},
					},
				},
				ChartMappings: map[string]any{
					"image.tag":             "{{ .Release.Spec.Version }}",
					`annotations.nesto\.ca`: true,
				},
				DefaultChart: "default/chart",
				CacheDir:     "~/.cache/joy/does_not_exist",
				SetupHelmMock: func(mpr *helm.MockPullRenderer) {
					mpr.EXPECT().
						Pull(context.Background(), helm.PullOptions{
							ChartURL:  "default/chart",
							Version:   "",
							OutputDir: "~/.cache/joy/does_not_exist/default/chart",
						}).
						Return(nil)

					mpr.EXPECT().
						Render(context.Background(), helm.RenderOpts{
							Dst:         &stdout,
							ReleaseName: "app",
							ChartPath:   "~/.cache/joy/does_not_exist/default/chart/chart",
							Values: map[string]any{
								"env":         "qa",
								"version":     "v1.2.3",
								"image":       map[string]any{"tag": "v1.2.3"},
								"annotations": map[string]any{"nesto.ca": true},
							},
						}).
						Return(nil)
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			helmMock := helm.NewMockPullRenderer(ctrl)

			if tc.Params.SetupHelmMock != nil {
				tc.Params.SetupHelmMock(helmMock)
			}

			err := Render(context.Background(), RenderOpts{
				Env:           tc.Params.Env,
				Release:       tc.Params.Release,
				DefaultChart:  tc.Params.DefaultChart,
				CacheDir:      tc.Params.CacheDir,
				ChartMappings: tc.Params.ChartMappings,
				Catalog:       tc.Params.Catalog,
				IO:            io,
				Helm:          helmMock,
				Color:         false,
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
