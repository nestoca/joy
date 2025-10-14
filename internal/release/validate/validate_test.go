package validate

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/davidmdm/x/xfs"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/helm"
)

func TestValidateRelease(t *testing.T) {
	disallowPullRequest := v1alpha1.Environment{Spec: v1alpha1.EnvironmentSpec{Promotion: v1alpha1.Promotion{FromPullRequests: false}}}
	allowPullRequest := v1alpha1.Environment{Spec: v1alpha1.EnvironmentSpec{Promotion: v1alpha1.Promotion{FromPullRequests: true}}}

	cases := []struct {
		Name          string
		Release       *v1alpha1.Release
		ChartFS       *xfs.FSMock
		HelmSetup     func(*helm.PullRendererMock)
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
			ExpectedErr: "",
		},
		{
			Name:    "spec does not match schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": true}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
			},
			ExpectedErr: "hydrating values: unifying with chart schema: validating values: #values.hello: conflicting values string and true (mismatched types string and bool)",
		},
		{
			Name:    "values missing from spec",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
			},
			ExpectedErr: "hydrating values: unifying with chart schema: validating values: #values.hello: incomplete value string",
		},
		{
			Name:    "multiple errors",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"one": "one", "two": "two"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { one: 1, two: 2 }`), nil },
			},
			ExpectedErr: "" +
				"hydrating values: unifying with chart schema: validating values:\n" +
				"  - #values.one: conflicting values 1 and \"one\" (mismatched types int and string)\n" +
				"  - #values.two: conflicting values 2 and \"two\" (mismatched types int and string)",
		},
		{
			Name:    "render fails",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "" },
			},

			HelmSetup: func(mock *helm.PullRendererMock) {
				mock.RenderFunc = func(ctx context.Context, opts helm.RenderOpts) (string, error) {
					return "", errors.New("failed to render")
				}
			},
			ExpectedErr: "failed to render",
		},
		{
			Name:    "no schema and render fails",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return nil, os.ErrNotExist },
				DirNameFunc:  func() string { return "" },
			},
			HelmSetup: func(mock *helm.PullRendererMock) {
				mock.RenderFunc = func(ctx context.Context, opts helm.RenderOpts) (string, error) {
					return "", errors.New("failed to render")
				}
			},
			ExpectedErr: "failed to render",
		},
		{
			Name:    "fail to read schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "world"}}, Environment: &allowPullRequest},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return nil, errors.New("disk corrupted") },
			},
			ExpectedErr: "hydrating values: unifying with chart schema: reading values.cue: disk corrupted",
		},
		{
			Name:    "standard version with disallow promotion from pull requests",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Version: "1.0.0", Values: map[string]any{"hello": "world"}}, Environment: &disallowPullRequest, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			ExpectedErr: "",
		},
		{
			Name:          "non-standard version with disallow promotion from pull requests",
			Release:       &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Version: "1.0.0-rc.1+build.1"}, Environment: &disallowPullRequest, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}}},
			SkipReadCalls: true,
			ExpectedErr:   "invalid version: prerelease branches not allowed: 1.0.0-rc.1+build.1",
		},
		{
			Name:    "non-standard version but skip pre-release check",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Version: "2.133.0-pactbroker2.116.0", Values: map[string]any{"hello": "world"}}, Environment: &disallowPullRequest, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{SkipPreReleaseCheck: true}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: string }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			ExpectedErr: "",
		},
		{
			Name:          "failing values hydration",
			Release:       &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"key": "$ref(.Invalid.Prefix.Ref)"}}, Environment: &v1alpha1.Environment{}, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}}},
			ChartFS:       &xfs.FSMock{},
			SkipReadCalls: true,
			ExpectedErr:   `hydrating values: hydrating object values: only ".Environment.Spec.Values." prefix is supported for object interpolation, but found: .Invalid.Prefix.Ref`,
		},
		{
			Name: "hydrated values matching schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "$ref(.Environment.Spec.Values.hello)"}}, Environment: &v1alpha1.Environment{
				Spec: v1alpha1.EnvironmentSpec{
					Values: map[string]any{"hello": "world"},
				},
			}, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: "world" }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			ExpectedErr: "",
		},
		{
			Name: "hydrated values not matching schema",
			Release: &v1alpha1.Release{Spec: v1alpha1.ReleaseSpec{Values: map[string]any{"hello": "$ref(.Environment.Spec.Values.hello)"}}, Environment: &v1alpha1.Environment{
				Spec: v1alpha1.EnvironmentSpec{
					Values: map[string]any{"hello": "narnia"},
				},
			}, Project: &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}}},
			ChartFS: &xfs.FSMock{
				ReadFileFunc: func(string) ([]byte, error) { return []byte(`#values: { hello: "world" }`), nil },
				DirNameFunc:  func() string { return "." },
			},
			ExpectedErr: `hydrating values: unifying with chart schema: validating values: #values.hello: conflicting values "narnia" and "world"`,
		},
		{
			Name: "contains locked todos",
			Release: &v1alpha1.Release{
				Environment: &v1alpha1.Environment{},
				Project:     &v1alpha1.Project{Spec: v1alpha1.ProjectSpec{}},
				File: &yml.File{Tree: func() *yaml.Node {
					var node yaml.Node
					_ = yaml.Unmarshal([]byte(`{ value: !lock TODO }`), &node)
					return &node
				}()},
			},
			ExpectedErr:   "contains locked TODO",
			SkipReadCalls: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mock := new(helm.PullRendererMock)
			if tc.HelmSetup != nil {
				tc.HelmSetup(mock)
			}

			if tc.Release != nil {
				if tc.Release.File == nil {
					tc.Release.File = new(yml.File)
				}
				if tc.Release.File.Tree == nil {
					tc.Release.File.Tree = new(yaml.Node)
				}
			}

			err := ValidateRelease(context.Background(), ValidateReleaseParams{
				Release: tc.Release,
				Chart:   &helm.ChartFS{FS: tc.ChartFS},
				Helm:    mock,
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
