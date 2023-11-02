package promote_test

import (
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr/github"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/pkg/catalog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"os/exec"
	"testing"
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
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintUpdatingTargetRelease(gomock.Any(), gomock.Any(), gomock.Any(), false)
	promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
	promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
	promptProvider.EXPECT().PrintCompleted()

	// Load catalog
	loadOpts := catalog.LoadOpts{
		Dir:             ".",
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
	promotion := promote.NewPromotion(promptProvider, promote.NewShellGitProvider("."), &github.PullRequestProvider{}, &promote.FileSystemYamlWriter{})
	opts := promote.Opts{
		Catalog:   cat,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
	}
	prURL, err := promotion.Promote(opts)
	defer func() {
		if prURL != "" {
			closePR(t, prURL)
		}
	}()

	// Check results
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func closePR(t *testing.T, prURL string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "close", prURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error closing PR %s: %s", prURL, output)
	}
}