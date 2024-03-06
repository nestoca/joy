package validate

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/davidmdm/x/xfs"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/helm"
)

func TestValidateRelease(t *testing.T) {
	disallowPullRequest := v1alpha1.Environment{Spec: v1alpha1.EnvironmentSpec{Promotion: v1alpha1.Promotion{FromPullRequests: false}}}
	allowPullRequest := v1alpha1.Environment{Spec: v1alpha1.EnvironmentSpec{Promotion: v1alpha1.Promotion{FromPullRequests: true}}}

	cases := []struct {
		Name          string
		Release       *v1alpha1.Release
		ChartFS       *xfs.FSMock
		HelmSetup     func(*helm.MockPullRenderer)
		ExpectedErr   string
		SkipReadCalls bool
	}{
		{
			Name:    "spec matches schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			HelmSetup: func(mpr *helm.MockPullRenderer) {
				mpr.EXPECT().Render(gomock.Any(), gomock.Any()).Return(nil)
			},
			ExpectedErr: "",
		},
		{
			Name:    "spec does not match schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": true}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
			},
			ExpectedErr: "hello: conflicting values true and string (mismatched types bool and string)",
		},
		{
			Name:    "render fails",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "" },
			},

			HelmSetup: func(mpr *helm.MockPullRenderer) {
				mpr.EXPECT().Render(gomock.Any(), gomock.Any()).Return(errors.New("failed to render"))
			},
			ExpectedErr: "rendering chart: failed to render",
		},
		{
			Name:    "no schema and render fails",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return nil, os.ErrNotExist },
				DirNameFunc:  func() string { return "" },
			},
			HelmSetup: func(mpr *helm.MockPullRenderer) {
				mpr.EXPECT().Render(gomock.Any(), gomock.Any()).Return(errors.New("failed to render"))
			},
			ExpectedErr: "rendering chart: failed to render",
		},
		{
			Name:    "fail to read schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return nil, errors.New("disk corrupted") },
			},
			ExpectedErr: "reading schema file: disk corrupted",
		},
		{
			Name:    "standard version with disallow promotion from pull requests",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Version: "1.0.0", Values: map[string]any{"hello": "world"}}, Environment: &disallowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			HelmSetup: func(mpr *helm.MockPullRenderer) {
				mpr.EXPECT().Render(gomock.Any(), gomock.Any()).Return(nil)
			},
			ExpectedErr: "",
		},
		{
			Name:          "non-standard version with disallow promotion from pull requests",
			Release:       &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Version: "1.0.0-rc.1+build.1"}, Environment: &disallowPullRequest},
			SkipReadCalls: true,
			ExpectedErr:   "invalid version: pre-release branches not allowed: 1.0.0-rc.1+build.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockedPullRenderer := helm.NewMockPullRenderer(ctrl)
			if tc.HelmSetup != nil {
				tc.HelmSetup(mockedPullRenderer)
			}

			err := ValidateRelease(context.Background(), ValidateReleaseParams{
				Release: tc.Release,
				Chart:   &helm.Chart{FS: tc.ChartFS},
				Helm:    mockedPullRenderer,
			})

			if tc.SkipReadCalls {
				require.EqualError(t, err, tc.ExpectedErr)
				return
			}

			require.Len(t, tc.ChartFS.ReadFileCalls(), 1)
			require.Equal(t, "values.cue", tc.ChartFS.ReadFileCalls()[0].Name)

			if tc.ExpectedErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tc.ExpectedErr)
		})
	}
}
