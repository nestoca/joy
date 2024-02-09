//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package promote

import (
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
)

type PromptProvider interface {
	// SelectSourceEnvironment prompts user to select source environment to promote from.
	SelectSourceEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error)

	// SelectTargetEnvironment prompts user to select target environment to promote to.
	SelectTargetEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error)

	// SelectReleases prompts user to select releases to promote.
	SelectReleases(list *cross.ReleaseList) (*cross.ReleaseList, error)

	// SelectCreatingPromotionPullRequest prompts user to confirm whether to continue creating promotion pull request
	// or abort.
	SelectCreatingPromotionPullRequest() (string, error)

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

	// PrintPullRequestCreated prints message that a new promotion pull request was created.
	PrintPullRequestCreated(url string)

	// PrintCanceled prints message that promotion was canceled and no pull request was created.
	PrintCanceled()

	// PrintCompleted prints message that the whole promotion operation was completed.
	PrintCompleted()
}
