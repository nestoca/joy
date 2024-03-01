package promote_test

import (
	"encoding/json"
	"os/exec"
	"testing"

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
	simpleCommitTemplate        = "Commit: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
	simplePullRequestTemplate   = "PR: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
	simpleProjectRepositoryFunc = func(proj *v1alpha1.Project) string {
		return "owner/" + proj.Name
	}
	simpleProjectSourceDirFunc = func(proj *v1alpha1.Project) (string, error) {
		return "/dummy/projects/" + proj.Name, nil
	}
	simpleCommitsMetadataFunc = func(projectDir, from, to string) ([]*promote.CommitMetadata, error) {
		return []*promote.CommitMetadata{
			{
				Sha:     "sha1",
				Message: "commit message 1",
			},
			{
				Sha:     "sha2",
				Message: "commit message 2",
			},
		}, nil
	}
	simpleCommitsGitHubAuthorsFunc = func(proj *v1alpha1.Project, fromTag, toTag string) (map[string]string, error) {
		return nil, nil
	}
	simpleReleaseGitTagFunc = func(release *v1alpha1.Release) (string, error) {
		return "v" + release.Spec.Version, nil
	}
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
	promptProvider.EXPECT().SelectCreatingPromotionPullRequest().Return(promote.Ready, nil)
	promptProvider.EXPECT().ConfirmAutoMergePullRequest().Return(false, nil)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(catalog.LoadOpts{Dir: dir})
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// Trigger the prompt since we are not auto-merging in this test
	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir),
		&promote.FileSystemYamlWriter{}, simpleCommitTemplate, simplePullRequestTemplate, simpleProjectRepositoryFunc, simpleProjectSourceDirFunc,
		simpleCommitsMetadataFunc, simpleCommitsGitHubAuthorsFunc, simpleReleaseGitTagFunc)
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
	promptProvider.EXPECT().ConfirmCreatingPromotionPullRequest(true, false).Return(true, nil)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:             dir,
		SortEnvsByOrder: true,
	})
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir),
		&promote.FileSystemYamlWriter{}, simpleCommitTemplate, simplePullRequestTemplate, simpleProjectRepositoryFunc, simpleProjectSourceDirFunc,
		simpleCommitsMetadataFunc, simpleCommitsGitHubAuthorsFunc, simpleReleaseGitTagFunc)

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

func TestEnforceEnvironmentAllowAutoMerge(t *testing.T) {
	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:             dir,
		SortEnvsByOrder: true,
	})
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	// EMPHASIS --- DRAMA
	targetEnv.Spec.Promotion.AllowAutoMerge = false

	// Perform test
	promotion := promote.NewPromotion(nil, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir),
		&promote.FileSystemYamlWriter{}, simpleCommitTemplate, simplePullRequestTemplate, simpleProjectRepositoryFunc, simpleProjectSourceDirFunc,
		simpleCommitsMetadataFunc, simpleCommitsGitHubAuthorsFunc, simpleReleaseGitTagFunc)

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
	promptProvider.EXPECT().ConfirmCreatingPromotionPullRequest(false, true).Return(true, nil)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintDraftPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	dir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:             dir,
		SortEnvsByOrder: true,
	})
	assert.NoError(t, err)

	// Resolve source and target environments
	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "staging")
	assert.NoError(t, err)

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, "prod")
	assert.NoError(t, err)

	targetEnv.Spec.Promotion.AllowAutoMerge = true

	// Perform test
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider(dir), github.NewPullRequestProvider(dir),
		&promote.FileSystemYamlWriter{}, simpleCommitTemplate, simplePullRequestTemplate, simpleProjectRepositoryFunc, simpleProjectSourceDirFunc,
		simpleCommitsMetadataFunc, simpleCommitsGitHubAuthorsFunc, simpleReleaseGitTagFunc)

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
