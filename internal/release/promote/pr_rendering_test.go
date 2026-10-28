package promote

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
)

func TestInjectPullRequestLinks(t *testing.T) {
	template := `[#{{ .PullRequestNumber }}](https://github.com/{{ .Repository }}/pull/{{ .PullRequestNumber }})`
	repository := "acme/project"

	tests := []struct {
		name     string
		template string
		text     string
		expected string
	}{
		{
			name:     "empty",
			template: template,
			text:     "",
			expected: "",
		},
		{
			name:     "no pr number",
			template: template,
			text:     "no pr number",
			expected: "no pr number",
		},
		{
			name:     "just pr number",
			template: template,
			text:     "#123",
			expected: "[#123](https://github.com/acme/project/pull/123)",
		},
		{
			name:     "pr number in the middle",
			template: template,
			text:     "text #123 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text",
		},
		{
			name:     "multiple pr numbers on different lines",
			template: template,
			text:     "text #123 text\ntext #456 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text\n" +
				"text [#456](https://github.com/acme/project/pull/456) text",
		},
		{
			name:     "multiple pr numbers on the same line",
			template: template,
			text:     "text #123 text #456 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text [#456](https://github.com/acme/project/pull/456) text",
		},
		{
			name:     "non-pr numbers",
			template: template,
			text:     "text#123 text1#123 #123text",
			expected: "text#123 text1#123 #123text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := injectPullRequestLinks(repository, tt.text)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetReleaseInfo(t *testing.T) {
	cases := []struct {
		Name           string
		Cross          *cross.Release
		Source, Target *v1alpha1.Release
		Opts           PerformOpts
		Expectations   func(*testing.T, *ReleaseInfo, error)
	}{
		{
			Name:   "ValuesChanged should be false if no promotion file",
			Cross:  &cross.Release{ValuesInSync: false, PromotedFile: nil},
			Source: &v1alpha1.Release{Project: &v1alpha1.Project{}},
			Target: &v1alpha1.Release{},
			Opts: PerformOpts{
				infoProvider:  new(info.ProviderMock),
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.False(t, releaseInfo.ValuesChanged)
			},
		},
		{
			Name:   "ValuesChanged should be false if values are in sync",
			Cross:  &cross.Release{ValuesInSync: true, PromotedFile: new(yml.File)},
			Source: &v1alpha1.Release{Project: &v1alpha1.Project{}},
			Target: &v1alpha1.Release{},
			Opts: PerformOpts{
				infoProvider:  new(info.ProviderMock),
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.False(t, releaseInfo.ValuesChanged)
			},
		},
		{
			Name:   "ValuesChanged should be true if values are not in sync",
			Cross:  &cross.Release{ValuesInSync: false, PromotedFile: new(yml.File)},
			Source: &v1alpha1.Release{Project: &v1alpha1.Project{}},
			Target: &v1alpha1.Release{},
			Opts: PerformOpts{
				infoProvider:  new(info.ProviderMock),
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.True(t, releaseInfo.ValuesChanged)
			},
		},
		{
			Name:  "GetCommitsMetadata failure preserves version and tag fields",
			Cross: &cross.Release{},
			Source: &v1alpha1.Release{
				Project: &v1alpha1.Project{},
				Spec:    v1alpha1.ReleaseSpec{Version: "1.0.1"},
			},
			Target: &v1alpha1.Release{
				Spec: v1alpha1.ReleaseSpec{Version: "1.0.0"},
			},
			Opts: PerformOpts{
				infoProvider: &info.ProviderMock{
					GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
						return "api/v" + release.Spec.Version, nil
					},
					GetProjectSourceDirFunc: func(_ *v1alpha1.Project) (string, error) {
						return "/some/dir", nil
					},
					GetCommitsMetadataFunc: func(_, _, _ string) ([]*info.CommitMetadata, error) {
						return nil, errors.New("fatal: ambiguous argument 'api/v1.0.0..api/v1.0.1'")
					},
				},
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.NotNil(t, releaseInfo)
				require.Error(t, releaseInfo.Error)
				require.Equal(t, "api/v1.0.1", releaseInfo.Source.GitTag)
				require.Equal(t, "api/v1.0.0", releaseInfo.Target.GitTag)
				require.Equal(t, "1.0.1", releaseInfo.Source.DisplayVersion)
				require.Equal(t, "1.0.0", releaseInfo.Target.DisplayVersion)
			},
		},
		{
			Name:  "missing git tag hints at gitTagTemplate",
			Cross: &cross.Release{},
			Source: &v1alpha1.Release{
				Project: &v1alpha1.Project{
					ProjectMetadata: v1alpha1.ProjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "brightwise"}},
					File:            &yml.File{Path: "projects/brightwise.yaml"},
				},
				Spec: v1alpha1.ReleaseSpec{Version: "0.18.2"},
			},
			Target: &v1alpha1.Release{
				Spec: v1alpha1.ReleaseSpec{Version: "0.18.1"},
			},
			Opts: PerformOpts{
				infoProvider: &info.ProviderMock{
					GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
						return "v" + release.Spec.Version, nil
					},
					GetProjectSourceDirFunc: func(_ *v1alpha1.Project) (string, error) {
						return "/some/dir", nil
					},
					GetCommitsMetadataFunc: func(_, _, _ string) ([]*info.CommitMetadata, error) {
						return nil, fmt.Errorf("%w: v0.18.1", info.ErrGitTagNotFound)
					},
				},
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.ErrorIs(t, releaseInfo.Error, info.ErrGitTagNotFound)
				require.ErrorContains(t, releaseInfo.Error, "gitTagTemplate")
				require.ErrorContains(t, releaseInfo.Error, "projects/brightwise.yaml")
			},
		},
		{
			Name:  "GetCommitsGitHubAuthors failure preserves version, tag, and commit list",
			Cross: &cross.Release{},
			Source: &v1alpha1.Release{
				Project: &v1alpha1.Project{},
				Spec:    v1alpha1.ReleaseSpec{Version: "1.0.1"},
			},
			Target: &v1alpha1.Release{
				Spec: v1alpha1.ReleaseSpec{Version: "1.0.0"},
			},
			Opts: PerformOpts{
				infoProvider: &info.ProviderMock{
					GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
						return "api/v" + release.Spec.Version, nil
					},
					GetProjectSourceDirFunc: func(_ *v1alpha1.Project) (string, error) {
						return "/some/dir", nil
					},
					GetCommitsMetadataFunc: func(_, _, _ string) ([]*info.CommitMetadata, error) {
						return []*info.CommitMetadata{
							{Sha: "abc1234", Author: "Alice", Message: "feat: add thing"},
						}, nil
					},
					GetCommitsGitHubAuthorsFunc: func(_ *v1alpha1.Project, _, _ string) (map[string]string, error) {
						return nil, errors.New("getting commits GitHub authors: HTTP 422")
					},
				},
				linksProvider: new(links.ProviderMock),
			},
			Expectations: func(t *testing.T, releaseInfo *ReleaseInfo, err error) {
				require.NoError(t, err)
				require.NotNil(t, releaseInfo)
				require.Error(t, releaseInfo.Error)
				require.Equal(t, "api/v1.0.1", releaseInfo.Source.GitTag)
				require.Equal(t, "api/v1.0.0", releaseInfo.Target.GitTag)
				require.Equal(t, "1.0.1", releaseInfo.Source.DisplayVersion)
				require.Equal(t, "1.0.0", releaseInfo.Target.DisplayVersion)
				require.Len(t, releaseInfo.Commits, 1)
				require.Equal(t, "abc1234", releaseInfo.Commits[0].Sha)
				require.Equal(t, "Alice", releaseInfo.Commits[0].Author)
				require.Empty(t, releaseInfo.Commits[0].GitHubAuthor)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			actual, err := getReleaseInfo(tc.Cross, tc.Source, tc.Target, tc.Opts)
			tc.Expectations(t, actual, err)
		})
	}
}
