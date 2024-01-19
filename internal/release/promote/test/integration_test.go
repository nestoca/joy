package promote_test

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr/github"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestPromoteAllReleasesFromStagingToProd(t *testing.T) {
	// Create mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	promptProvider := promote.NewMockPromptProvider(ctrl)

	// Set expectations
	promptProvider.EXPECT().SelectReleases(gomock.Any()).DoAndReturn(func(list *cross.ReleaseList) (*cross.ReleaseList, error) { return list, nil })
	promptProvider.EXPECT().PrintStartPreview()
	promptProvider.EXPECT().PrintReleasePreview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintReleasePreview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintEndPreview()
	promptProvider.EXPECT().ConfirmCreatingPromotionPullRequest().Return(true, nil)
	promptProvider.EXPECT().ConfirmAutoMergePullRequest().Return(false, nil)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	// Load catalog
	loadOpts := catalog.LoadOpts{
		Dir:             dir,
		LoadEnvs:        true,
		LoadReleases:    true,
		SortEnvsByOrder: true,
	}

	cat, err := catalog.Load(loadOpts)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// Trigger the prompt since we are not automerging in this test
	targetEnv.Spec.Promotion.FromPullRequests = true

	// Perform test
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir), &promote.FileSystemYamlWriter{})
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
	// Create mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	promptProvider := promote.NewMockPromptProvider(ctrl)

	// Set expectations
	promptProvider.EXPECT().SelectReleases(gomock.Any()).DoAndReturn(func(list *cross.ReleaseList) (*cross.ReleaseList, error) { return list, nil })
	promptProvider.EXPECT().PrintStartPreview()
	promptProvider.EXPECT().PrintReleasePreview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintReleasePreview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintEndPreview()
	promptProvider.EXPECT().ConfirmCreatingPromotionPullRequest().Return(true, nil)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	// Load catalog
	loadOpts := catalog.LoadOpts{
		Dir:             dir,
		LoadEnvs:        true,
		LoadReleases:    true,
		SortEnvsByOrder: true,
	}

	cat, err := catalog.Load(loadOpts)
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// Perform test
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir), &promote.FileSystemYamlWriter{})

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

	require.Equal(t, []string{"auto-merge"}, getPullRequestLabels(t, dir, prURL))
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
