package promote

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
)

const messageTemplate = `{{- $sourceEnv := .SourceEnvironment.Name -}}
{{- $targetEnv := .TargetEnvironment.Name -}}
{{- if eq (len .Releases) 1 -}}
{{- with (index .Releases 0) -}}
Promote {{ .Name }} {{ .Target.DisplayVersion }} -> {{ .Source.DisplayVersion }} ({{ $sourceEnv }} -> {{ $targetEnv }})
{{- end -}}
{{- else -}}
Promote {{ len .Releases }} releases ({{ $sourceEnv }} -> {{ $targetEnv }})
{{- end }}
# Promotions

| Release | Diff | Source: {{ $sourceEnv }} | Argo CD | Datadog | Target: {{ $targetEnv }} | Argo CD | Datadog |
|---|---|---|---|---|---|---|---|
{{ range .Releases -}}
{{- $diff := "" -}}
{{- if .Target -}}
{{- $diff = printf "[Diff](github/%s/v%s...v%s)" .Repository .Target.Spec.Version .Source.Spec.Version -}}
{{- else  -}}
{{- $diff = printf "[Entire](github/%s/releases/tag/%s)" .Repository .Source.Spec.Version -}}
{{- end -}}
{{- $formatArgoCD := "[Argo CD](argocd/%s/%s)" -}}
{{- $sourceArgoCD := printf $formatArgoCD $sourceEnv .Name -}}
{{- $targetArgoCD := printf $formatArgoCD $targetEnv .Name -}}
{{- $formatDatadog := "[Datadog](datadog/%s/%s)" -}}
{{- $sourceDatadog := printf $formatDatadog $sourceEnv .Name -}}
{{- $targetDatadog := printf $formatDatadog $targetEnv .Name -}}
| {{ .Name }} | {{ $diff }} | {{ .Source.Release.Spec.Version }} | {{ $sourceArgoCD }} | {{ $sourceDatadog }} | {{ .Target.Release.Spec.Version }} | {{ $targetArgoCD }} | {{ $targetDatadog }} |
{{ end }}{{ range .Releases }}
# {{ .Name }} commits

| SHA | Author | Message |
| --- | --- | --- |
{{ $repository := .Repository -}}
{{- range .Commits -}}
| [github/{{ $repository }}/{{ .Sha }}]({{ .ShortSha }}) | @{{ .GitHubAuthor }} | {{ .Message }} |
{{ end }}
{{- end }}
Variable1: {{ .Variables.variable1 }}
`

const expectedMessage = `Promote 2 releases (staging -> production)
# Promotions

| Release | Diff | Source: staging | Argo CD | Datadog | Target: production | Argo CD | Datadog |
|---|---|---|---|---|---|---|---|
| project1 | [Diff](github/acme/project1/v1.2.3...v1.2.4) | 1.2.4 | [Argo CD](argocd/staging/project1) | [Datadog](datadog/staging/project1) | 1.2.3 | [Argo CD](argocd/production/project1) | [Datadog](datadog/production/project1) |
| project2 | [Diff](github/acme/project2/v1.2.3...v1.2.4) | 1.2.4 | [Argo CD](argocd/staging/project2) | [Datadog](datadog/staging/project2) | 1.2.3 | [Argo CD](argocd/production/project2) | [Datadog](datadog/production/project2) |

# project1 commits

| SHA | Author | Message |
| --- | --- | --- |
| [github/acme/project1/1234567890123456789012345678901234567890](1234567) | @gh-author1 | commit message 1 |
| [github/acme/project1/4567890123456789012345678901234567890123](4567890) | @gh-author2 | commit message 2 |

# project2 commits

| SHA | Author | Message |
| --- | --- | --- |
| [github/acme/project2/1234567890123456789012345678901234567890](1234567) | @gh-author1 | commit message 1 |
| [github/acme/project2/4567890123456789012345678901234567890123](4567890) | @gh-author2 | commit message 2 |

Variable1: value1
`

func TestRenderMessage(t *testing.T) {
	commits := []*CommitInfo{
		{
			Sha:          "1234567890123456789012345678901234567890",
			ShortSha:     "1234567",
			Author:       "author1",
			GitHubAuthor: "gh-author1",
			Message:      "commit message 1",
		},
		{
			Sha:          "4567890123456789012345678901234567890123",
			ShortSha:     "4567890",
			Author:       "author2",
			GitHubAuthor: "gh-author2",
			Message:      "commit message 2",
		},
	}
	info := &PromotionInfo{
		SourceEnvironment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "staging"}},
		TargetEnvironment: &v1alpha1.Environment{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "production"}},
		Releases: []*ReleaseInfo{
			{
				Name: "project1",
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Repository: "project1",
					},
				},
				Repository: "acme/project1",
				Source: EnvironmentReleaseInfo{
					Release: &v1alpha1.Release{
						Spec: v1alpha1.ReleaseSpec{
							Version: "1.2.4",
						},
					},
					GitTag: "v1.2.4",
				},
				Target: EnvironmentReleaseInfo{
					Release: &v1alpha1.Release{
						Spec: v1alpha1.ReleaseSpec{
							Version: "1.2.3",
						},
					},
					GitTag: "v1.2.3",
				},
				Commits: commits,
			},
			{
				Name: "project2",
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Repository: "project2",
					},
				},
				Repository: "acme/project2",
				Source: EnvironmentReleaseInfo{
					Release: &v1alpha1.Release{
						Spec: v1alpha1.ReleaseSpec{
							Version: "1.2.4",
						},
					},
					GitTag: "v1.2.4",
				},
				Target: EnvironmentReleaseInfo{
					Release: &v1alpha1.Release{
						Spec: v1alpha1.ReleaseSpec{
							Version: "1.2.3",
						},
					},
					GitTag: "v1.2.3",
				},
				Commits: commits,
			},
		},
		Variables: map[string]string{
			"variable1": "value1",
		},
	}

	message, err := renderMessage(messageTemplate, info)
	require.NoError(t, err)
	require.Equal(t, expectedMessage, message)
}

func TestGetReviewers(t *testing.T) {
	require.Equal(
		t,
		[]string{"john"},
		getReviewers(&PromotionInfo{
			Releases: []*ReleaseInfo{
				{Reviewers: []string{"john"}},
				{Error: errors.New("error")},
			},
		}),
	)
}
