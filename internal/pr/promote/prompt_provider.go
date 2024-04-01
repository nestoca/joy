package promote

//go:generate moq -stub -out ./prompt_provider_mock.go . PromptProvider
type PromptProvider interface {
	// WhetherToCreateMissingPullRequest prompts user to create a pull request for current branch.
	WhetherToCreateMissingPullRequest() (bool, error)

	// WhichEnvironmentToPromoteTo prompts user to select environment to auto-promote builds of pull request to.
	// If empty string is returned, user opted to disable auto-promotion.
	WhichEnvironmentToPromoteTo(environments []string, preSelectedEnv string) (string, error)

	// ConfirmDisablingPromotionOnOtherPullRequest prompts user to confirm it is ok to disable auto-promotion of another
	// pull request having same target environment.
	ConfirmDisablingPromotionOnOtherPullRequest(branch, env string) (bool, error)

	// PrintBranchDoesNotSupportAutoPromotion prints message that master/main branch cannot be promoted.
	PrintBranchDoesNotSupportAutoPromotion(branch string)

	// PrintNotCreatingPullRequest prints message that user opted not to create pull request.
	PrintNotCreatingPullRequest()

	// PrintPromotionAlreadyConfigured prints message that given branch is configured for auto-promotion to given
	// environment.
	PrintPromotionAlreadyConfigured(branch, env string)

	// PrintPromotionNotConfigured prints message that promotion was not configured.
	PrintPromotionNotConfigured(branch, env string)

	// PrintPromotionConfigured prints message that promotion was correctly configured.
	PrintPromotionConfigured(branch, env string)

	// PrintPromotionDisabled prints message that promotion was disabled.
	PrintPromotionDisabled(branch string)
}
