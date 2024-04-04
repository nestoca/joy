package promote_test

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/yml"

	"github.com/nestoca/joy/internal/info"

	"github.com/nestoca/joy/internal/github"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/pkg/catalog"
)

var (
	simpleCommitTemplate      = "Commit: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
	simplePullRequestTemplate = "PR: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
)

func newMockInfoProvider() *info.ProviderMock {
	return &info.ProviderMock{
		GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
			return "v1.0.0", nil
		},
		GetProjectRepositoryFunc: func(project *v1alpha1.Project) string {
			return "owner/project"
		},
		GetProjectSourceDirFunc: func(project *v1alpha1.Project) (string, error) {
			return "/dummp/projects/project", nil
		},
		GetCommitsMetadataFunc: func(projectDir, fromTag, toTag string) ([]*info.CommitMetadata, error) {
			return nil, nil
		},
		GetCommitsGitHubAuthorsFunc: func(project *v1alpha1.Project, fromTag, toTag string) (map[string]string, error) {
			return nil, nil
		},
	}
}

func TestPromoteAllReleasesFromStagingToProd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	promptProvider := new(promote.PromptProviderMock)
	promptProvider.SelectReleasesFunc = func(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
		return list, nil
	}
	defer func() {
		require.Len(t, promptProvider.PrintUpdatingTargetReleaseCalls(), 2)
		for _, call := range promptProvider.PrintUpdatingTargetReleaseCalls() {
			require.Equal(t, false, call.IsCreating)
		}
	}()

	infoProvider := newMockInfoProvider()
	linksProvider := new(links.ProviderMock)

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(dir, nil)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// Trigger the prompt since we are not auto-merging in this test
	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.Promotion{
		PromptProvider:      promptProvider,
		GitProvider:         promote.NewShellGitProvider(dir),
		PullRequestProvider: github.NewPullRequestProvider(dir),
		YamlWriter:          yml.DiskWriter,
		CommitTemplate:      simpleCommitTemplate,
		PullRequestTemplate: simplePullRequestTemplate,
		InfoProvider:        infoProvider,
		LinksProvider:       linksProvider,
	}
	opts := promote.Opts{
		Catalog:   cat,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
	}

	prURL, err := promotion.Promote(opts)
	defer closePR(t, dir, prURL)

	require.NoError(t, err)
	require.NotEmpty(t, prURL)
}

func TestPromoteAutoMergeFromStagingToProd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	promptProvider := &promote.PromptProviderMock{
		SelectReleasesFunc: func(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
			return list, nil
		},
		ConfirmCreatingPromotionPullRequestFunc: func(autoMerge, draft bool) (bool, error) {
			return true, nil
		},
	}
	defer func() {
		require.Len(t, promptProvider.PrintUpdatingTargetReleaseCalls(), 2)
		for _, call := range promptProvider.PrintUpdatingTargetReleaseCalls() {
			require.Equal(t, false, call.IsCreating)
		}
		require.Len(t, promptProvider.ConfirmCreatingPromotionPullRequestCalls(), 1)
		require.Equal(t, true, promptProvider.ConfirmCreatingPromotionPullRequestCalls()[0].AutoMerge)
		require.Equal(t, false, promptProvider.ConfirmCreatingPromotionPullRequestCalls()[0].Draft)
	}()

	infoProvider := newMockInfoProvider()
	linksProvider := new(links.ProviderMock)

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(dir, nil)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.Promotion{
		PromptProvider:      promptProvider,
		GitProvider:         promote.NewShellGitProvider(dir),
		PullRequestProvider: github.NewPullRequestProvider(dir),
		YamlWriter:          yml.DiskWriter,
		CommitTemplate:      simpleCommitTemplate,
		PullRequestTemplate: simplePullRequestTemplate,
		InfoProvider:        infoProvider,
		LinksProvider:       linksProvider,
	}
	opts := promote.Opts{
		Catalog:   cat,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
		AutoMerge: true,
	}

	prURL, err := promotion.Promote(opts)
	defer closePR(t, dir, prURL)

	require.NoError(t, err)
	require.NotEmpty(t, prURL)

	require.Equal(t, []string{"auto-merge", "environment:prod", "release:bar", "release:foo"}, getPullRequestLabels(t, dir, prURL))
}

func TestEnforceEnvironmentAllowAutoMerge(t *testing.T) {
	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(dir, nil)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// EMPHASIS --- DRAMA
	targetEnv.Spec.Promotion.AllowAutoMerge = false

	// Perform test
	promotion := promote.Promotion{
		PromptProvider:      nil,
		GitProvider:         promote.NewShellGitProvider(dir),
		PullRequestProvider: github.NewPullRequestProvider(dir),
		YamlWriter:          yml.DiskWriter,
		CommitTemplate:      simpleCommitTemplate,
		PullRequestTemplate: simplePullRequestTemplate,
		InfoProvider:        nil,
		LinksProvider:       nil,
	}
	opts := promote.Opts{
		Catalog:   cat,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
		AutoMerge: true,
	}

	prURL, err := promotion.Promote(opts)
	require.Empty(t, prURL)
	require.EqualError(t, err, "auto-merge is not allowed for target environment prod")
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

	return labels
}

func closePR(t *testing.T, dir, prURL string) {
	t.Helper()

	cmd := exec.Command("gh", "pr", "close", prURL)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error closing PR %s: %s", prURL, output)
	}
}

func TestDraftPromoteFromStagingToProd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	promptProvider := &promote.PromptProviderMock{
		SelectReleasesFunc: func(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
			return list, nil
		},
		ConfirmCreatingPromotionPullRequestFunc: func(autoMerge, draft bool) (bool, error) {
			return true, nil
		},
	}
	defer func() {
		require.Len(t, promptProvider.PrintUpdatingTargetReleaseCalls(), 2)
		for _, call := range promptProvider.PrintUpdatingTargetReleaseCalls() {
			require.Equal(t, false, call.IsCreating)
		}
		require.Len(t, promptProvider.ConfirmCreatingPromotionPullRequestCalls(), 1)
		require.Equal(t, false, promptProvider.ConfirmCreatingPromotionPullRequestCalls()[0].AutoMerge)
		require.Equal(t, true, promptProvider.ConfirmCreatingPromotionPullRequestCalls()[0].Draft)
	}()

	infoProvider := newMockInfoProvider()
	linksProvider := new(links.ProviderMock)

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(dir, nil)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.Promotion{
		PromptProvider:      promptProvider,
		GitProvider:         promote.NewShellGitProvider(dir),
		PullRequestProvider: github.NewPullRequestProvider(dir),
		YamlWriter:          yml.DiskWriter,
		CommitTemplate:      simpleCommitTemplate,
		PullRequestTemplate: simplePullRequestTemplate,
		InfoProvider:        infoProvider,
		LinksProvider:       linksProvider,
	}
	opts := promote.Opts{
		Catalog:   cat,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
		Draft:     true,
	}

	prURL, err := promotion.Promote(opts)
	defer closePR(t, dir, prURL)

	require.NoError(t, err)
	require.NotEmpty(t, prURL)

	require.Equal(t, true, isPullRequestDraft(t, dir, prURL))
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
