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
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/testutils"
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

func executeReleasePromoteCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(dir, nil)
	require.NoError(t, err)

	cfg, err := config.Load(dir, dir)
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
