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
	// TODO AYA:
	// Add tests where the release version and promotion rules conflict.

	cases := []struct {
		Name        string
		Release     *v1alpha1.Release
		ChartFS     *xfs.FSMock
		HelmSetup   func(*helm.MockPullRenderer)
		ExpectedErr string
	}{
		{
			Name:    "spec matches schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}},
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
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": true}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
			},
			ExpectedErr: "hello: conflicting values true and string (mismatched types bool and string)",
		},
		{
			Name:    "render fails",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}},
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
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}},
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
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return nil, errors.New("disk corrupted") },
			},
			ExpectedErr: "reading schema file: disk corrupted",
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
