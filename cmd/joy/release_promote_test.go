package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestPromoteAllReleasesFromStagingToProd(t *testing.T) {
	dir, output, err := executeReleasePromoteCommand(t, NewReleasePromoteCmd(PromoteParams{PreRunConfigs: make(PreRunConfigs)}),
		"--source", "staging",
		"--target", "prod",
		"--no-prompt",
		"--draft",
		"--template-var", "triggeredBy=john.doe",
		"foo")
	require.NoError(t, err)

	prURLRegex := regexp.MustCompile(`Created draft pull request: https://github.com/nestoca/joy-release-promote-test/pull/([0-9]+)`)
	match := prURLRegex.FindStringSubmatch(output)
	require.NotNil(t, match, "Pull request URL should be specified in output:\n"+output+"\n---")
	prURL := match[1]
	defer closePR(t, dir, prURL)

	branchRegex := regexp.MustCompile(`Committed and pushed new branch (promote-foo-from-staging-to-prod-([0-9a-f-]+)) with message:`)
	match = branchRegex.FindStringSubmatch(output)
	require.NotNil(t, match, "Branch name should be specified in output:\n"+output+"\n---")
	branch := match[1]

	require.NoError(t, git.Checkout(dir, branch))
	actual, err := os.ReadFile(filepath.Join(dir, "environments/prod/foo.yaml"))
	require.NoError(t, err)
	expected := `apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: foo
spec:
  project: foo
  version: '1.2.4'
  values:
    replicas: !lock 2
    env:
      PORT: 8080
      NEW_VAR: value
`
	require.Equal(t, expected, string(actual))
	require.Equal(t, true, isPullRequestDraft(t, dir, prURL), "PR should be a draft")

	actualBody := getPullRequestBody(t, dir, prURL)
	expectedBody := `Triggered by: john.doe

# Promotions
| Release | Source: staging | Target: prod |
|:---|:---|:---|
| foo | [1.2.4](https://github.com/nestoca/joy-release-promote-test/releases/tag/v1.2.4)<br> [Logs](https://app.datadoghq.com/logs?query=env%3Astaging%20service%3Afoo&cols=env%2Cservice%2Cpod_name) | [1.2.3](https://github.com/nestoca/joy-release-promote-test/releases/tag/v1.2.3)<br> [Logs](https://app.datadoghq.com/logs?query=env%3Aprod%20service%3Afoo&cols=env%2Cservice%2Cpod_name) |

# Upgrade commits for foo

[Compare](https://github.com/nestoca/joy-release-promote-test/compare/v1.2.3...v1.2.4) v1.2.3...v1.2.4 on GitHub

| Message | Author | Commit |
|:---|:---|:---|
| Dummy change made by nestobot | @nestobot | [bedd22e](https://github.com/nestoca/joy-release-promote-test/commit/bedd22e73a04141c121bc70ac138e064b01b8fb2) |`
	require.Equal(t, expectedBody, actualBody)

	require.Equal(t, []string{"environment:prod", "release:foo"}, getPullRequestLabels(t, dir, prURL))
}

func TestAutoMerge(t *testing.T) {
	dir, output, err := executeReleasePromoteCommand(t, NewReleasePromoteCmd(PromoteParams{PreRunConfigs: make(PreRunConfigs)}),
		"--source", "staging",
		"--target", "dev",
		"--no-prompt",
		"--auto-merge",
		"foo")
	require.NoError(t, err)

	prURLRegex := regexp.MustCompile(`Created pull request: https://github.com/nestoca/joy-release-promote-test/pull/([0-9]+)`)
	match := prURLRegex.FindStringSubmatch(output)
	require.NotNil(t, match, "Pull request URL should be specified in output:\n"+output+"\n---")
	prURL := match[1]
	defer closePR(t, dir, prURL)

	require.Equal(t, []string{"auto-merge", "environment:dev", "release:foo"}, getPullRequestLabels(t, dir, prURL))
}

func TestDisallowedAutoMerge(t *testing.T) {
	_, _, err := executeReleasePromoteCommand(t, NewReleasePromoteCmd(PromoteParams{PreRunConfigs: make(PreRunConfigs)}),
		"--source", "staging",
		"--target", "prod",
		"--no-prompt",
		"--auto-merge",
		"foo")
	require.NotNil(t, err)
	require.Equal(t, "auto-merge is not allowed for target environment prod", err.Error())
}

func TestViewGitLog(t *testing.T) {
	cases := []struct {
		Name              string
		Expected          string
		InfoProviderSetup func(*info.ProviderMock) func(*testing.T)
	}{
		{
			Name:     "no commits for service",
			Expected: "--- test\nno commits found in repo",
			InfoProviderSetup: func(mock *info.ProviderMock) func(*testing.T) {
				return func(t *testing.T) {}
			},
		},
		{
			Name: "with git log information",
			Expected: "" +
				"--- test\n" +
				"sha-1 gh-author-1 msg-1\n" +
				"sha-2 gh-author-2 msg-2",
			InfoProviderSetup: func(mock *info.ProviderMock) func(*testing.T) {
				mock.GetCommitsMetadataFunc = func(projectDir, fromTag, toTag string) ([]*info.CommitMetadata, error) {
					return []*info.CommitMetadata{
						{Sha: "sha-1", Message: "msg-1"},
						{Sha: "sha-2", Message: "msg-2"},
					}, nil
				}

				mock.GetCommitsGitHubAuthorsFunc = func(project *v1alpha1.Project, fromTag, toTag string) (map[string]string, error) {
					return map[string]string{
						"sha-1": "gh-author-1",
						"sha-2": "gh-author-2",
					}, nil
				}

				return func(t *testing.T) {
					require.Len(t, mock.GetCommitsMetadataCalls(), 1)
					require.Len(t, mock.GetCommitsGitHubAuthorsCalls(), 1)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			info := new(info.ProviderMock)

			if tc.InfoProviderSetup != nil {
				defer tc.InfoProviderSetup(info)(t)
			}

			cmd := NewReleasePromoteCmd(PromoteParams{
				Info:        info,
				Links:       &links.ProviderMock{},
				Git:         &promote.GitProviderMock{},
				PullRequest: nil,
				Prompt: &promote.PromptProviderMock{
					SelectPromotionActionFunc: func() func() (string, error) {
						first := true
						return func() (string, error) {
							if first {
								first = false
								return promote.ViewGitLog, nil
							}
							return promote.Cancel, nil
						}
					}(),
				},
				Writer:        nil,
				PreRunConfigs: map[*cobra.Command]PreRunConfig{},
			})

			var buffer bytes.Buffer

			cmd.SetOut(&buffer)
			cmd.SetArgs([]string{"--source=source", "--target=target", "test"})

			environments := []*v1alpha1.Environment{
				{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "source"}},
				{
					EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: "target"},
					Spec: v1alpha1.EnvironmentSpec{
						Promotion: v1alpha1.Promotion{
							FromEnvironments: []string{"source"},
						},
					},
				},
			}

			ctx := catalog.ToContext(context.Background(), &catalog.Catalog{
				Environments: environments,
				Releases: cross.ReleaseList{
					Environments: environments,
					Items: []*cross.Release{
						{
							Name: "test",
							Releases: []*v1alpha1.Release{
								{
									ReleaseMetadata: v1alpha1.ReleaseMetadata{},
									Spec:            v1alpha1.ReleaseSpec{},
									File:            &yml.File{},
									Project:         &v1alpha1.Project{},
									Environment:     environments[0],
								},
								{
									ReleaseMetadata: v1alpha1.ReleaseMetadata{},
									Spec:            v1alpha1.ReleaseSpec{},
									File:            &yml.File{},
									Project:         &v1alpha1.Project{},
									Environment:     environments[1],
								},
							},
						},
					},
				},
			})

			ctx = config.ToContext(ctx, &config.Config{})

			cmd.SetContext(ctx)

			require.NoError(t, cmd.Execute())

			actual := buffer.String()
			actual = stripansi.Strip(actual)
			actual = strings.TrimSpace(actual)

			require.Equal(t, tc.Expected, actual)
		})
	}
}

func executeReleasePromoteCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(context.Background(), dir, nil)
	require.NoError(t, err)

	cfg, err := config.Load(context.Background(), dir, dir)
	require.NoError(t, err)

	ctx := config.ToContext(catalog.ToContext(context.Background(), cat), cfg)

	var buffer bytes.Buffer
	cmd.SetOut(&buffer)
	cmd.SetArgs(args)
	cmd.SetContext(ctx)

	err = cmd.Execute()
	return dir, stripansi.Strip(buffer.String()), err
}

func closePR(t *testing.T, dir, prURL string) {
	t.Helper()

	cmd := exec.Command("gh", "pr", "close", "--delete-branch", prURL)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error closing PR %s: %s", prURL, output)
	}
}

func getPullRequestLabels(t *testing.T, dir, url string) []string {
	cmd := exec.Command("gh", "pr", "view", "--json", "labels", url)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to get pr labels: %q", output)

	var result struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	require.NoError(t, json.Unmarshal(output, &result), "failed to unmarshal input: %q", output)

	labels := make([]string, len(result.Labels))
	for i, value := range result.Labels {
		labels[i] = value.Name
	}
	slices.Sort(labels)

	return labels
}

func isPullRequestDraft(t *testing.T, dir, url string) bool {
	cmd := exec.Command("gh", "pr", "view", "--json", "isDraft", url)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to get PR draft state: %q", output)

	var result struct {
		IsDraft bool `json:"isDraft"`
	}
	require.NoError(t, json.Unmarshal(output, &result), "failed to unmarshal input: %q", output)

	return result.IsDraft
}

func getPullRequestBody(t *testing.T, dir, url string) string {
	cmd := exec.Command("gh", "pr", "view", "--json", "body", url)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to get PR body: %q", output)

	var result struct {
		Body string `json:"body"`
	}
	require.NoError(t, json.Unmarshal(output, &result), "failed to unmarshal input: %q", output)

	return result.Body
}
