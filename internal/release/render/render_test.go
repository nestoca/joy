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
								"image":   map[string]any{"tag": "v1.2.3"},
								"common":  map[string]any{"annotations": map[string]any{"nesto.ca/deployed-by": "joy"}},
							},
						}).
						Return(nil)
				},
			},
			ExpectedOut: "error hydrating values: template: :4: unterminated raw quoted string\nfallback to raw release.spec.values\n",
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
								"image":   map[string]any{"tag": "v1.2.3"},
								"common":  map[string]any{"annotations": map[string]any{"nesto.ca/deployed-by": "joy"}},
							},
						}).
						Return(errors.New("bebop"))
				},
			},
			ExpectedError: "rendering chart: bebop",
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
				Env:          tc.Params.Env,
				Release:      tc.Params.Release,
				DefaultChart: tc.Params.DefaultChart,
				CacheDir:     tc.Params.CacheDir,
				Catalog:      tc.Params.Catalog,
				IO:           io,
				Helm:         helmMock,
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
