//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package promote

type Prompt interface {
	// WhetherToCreateMissingPullRequest prompts user to create a pull request for current branch.
	WhetherToCreateMissingPullRequest() (bool, error)

	// WhichEnvironmentToPromoteTo prompts user to select environment to auto-promote builds of pull request to.
	// If empty string is returned, user opted to disable auto-promotion.
	WhichEnvironmentToPromoteTo(environments []string, preSelectedEnv string) (string, error)

	// PrintMasterBranchPromotion prints message that master/main branch cannot be promoted.
	PrintMasterBranchPromotion()

	// PrintNotCreatingPullRequest prints message that user opted not to create pull request.
	PrintNotCreatingPullRequest()

	// PrintPromotionConfigured prints message that promotion was correctly configured.
	PrintPromotionConfigured(branch string, env string)

	// PrintPromotionDisabled prints message that promotion was disabled.
	PrintPromotionDisabled(branch string)
}
