package promote

import (
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
)

//go:generate moq -stub -out ./prompt_provider_mock.go . PromptProvider
type PromptProvider interface {
	// SelectSourceEnvironment prompts user to select source environment to promote from.
	SelectSourceEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error)

	// SelectTargetEnvironment prompts user to select target environment to promote to.
	SelectTargetEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error)

	// SelectReleases prompts user to select releases to promote.
	SelectReleases(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error)

	// ConfirmCreatingPromotionPullRequest prompts user to confirm whether to continue creating promotion pull request
	// or abort.
	ConfirmCreatingPromotionPullRequest(autoMerge, draft bool) (bool, error)

	// SelectPromotionAction prompts user to select state of promotion PR
	// or abort.
	SelectPromotionAction() (string, error)

	// ConfirmAutoMergePullRequest prompts user to confirm whether to auto-merge promotion PR or not
	ConfirmAutoMergePullRequest() (bool, error)

	// PrintNoPromotableReleasesFound prints message that no promotable releases were found for given
	// source and target environments or potentially none because release filtering was applied.
	PrintNoPromotableReleasesFound(releasesFiltered bool, sourceEnv *v1alpha1.Environment, targetEnv *v1alpha1.Environment)

	// PrintNoPromotableEnvironmentFound prints message that no promotable environments were found
	// or potentially none because environment filtering was applied.
	PrintNoPromotableEnvironmentFound(environmentsFiltered bool)

	// PrintStartPreview starts the preview of release promotion diffs.
	PrintStartPreview()

	// PrintReleasePreview prints the diff for promotion of a given release.
	PrintReleasePreview(targetEnvName string, releaseName string, existingTargetFile, promotedFile *yml.File) error

	// PrintEndPreview ends the preview of release promotion diffs.
	PrintEndPreview()

	// PrintUpdatingTargetRelease prints message that target release file is being updated or created.
	PrintUpdatingTargetRelease(targetEnvName, releaseName, releaseFilePath string, isCreating bool)

	// PrintBranchCreated prints message that a new promotion branch was created and promotion changes were committed
	// and pushed.
	PrintBranchCreated(branchName, message string)

	// PrintDraftPullRequestCreated prints message that a new promotion draft pull request was created.
	PrintDraftPullRequestCreated(url string)

	// PrintPullRequestCreated prints message that a new promotion pull request was created.
	PrintPullRequestCreated(url string)

	// PrintCanceled prints message that promotion was canceled and no pull request was created.
	PrintCanceled()

	// PrintCompleted prints message that the whole promotion operation was completed.
	PrintCompleted()

	// PrintSelectedNonPromotableReleases prints message that non-promotable releases were selected.
	PrintSelectedNonPromotableReleases(invalidReleases, targetEnvName string)
}
