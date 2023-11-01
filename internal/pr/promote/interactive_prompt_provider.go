package promote

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"github.com/nestoca/joy/internal/style"
)

type InteractivePromptProvider struct{}

func (s *InteractivePromptProvider) WhetherToCreateMissingPullRequest() (bool, error) {
	prompt := &survey.Confirm{
		Message: "No pull request found for current branch, create one?",
		Default: true,
	}
	var shouldCreate bool
	err := survey.AskOne(prompt, &shouldCreate)
	if err != nil {
		return false, fmt.Errorf("prompting user to create pull request: %w", err)
	}
	return shouldCreate, nil
}

func (s *InteractivePromptProvider) WhichEnvironmentToPromoteTo(environments []string, preSelectedEnv string) (string, error) {
	none := "[none]"
	if preSelectedEnv == "" {
		preSelectedEnv = none
	}
	options := append([]string{none}, environments...)
	prompt := &survey.Select{
		Message: "Select environment to auto-promote builds of pull request to:",
		Options: options,
		Default: preSelectedEnv,
	}
	var selectedEnv string
	err := survey.AskOne(prompt, &selectedEnv)
	if err != nil {
		return "", fmt.Errorf("prompting user to select environment: %w", err)
	}
	if selectedEnv == none {
		selectedEnv = ""
	}
	return selectedEnv, nil
}

func (s *InteractivePromptProvider) ConfirmDisablingPromotionOnOtherPullRequest(branch, env string) (bool, error) {
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Another pull request for branch %s is already auto-promoting to %s environment, disable it?", style.Resource(branch), style.Resource(env)),
		Default: false,
	}
	var shouldDisable bool
	err := survey.AskOne(prompt, &shouldDisable)
	if err != nil {
		return false, fmt.Errorf("prompting user to confirm disabling promotion on other pull request: %w", err)
	}
	return shouldDisable, nil
}

func (s *InteractivePromptProvider) PrintBranchDoesNotSupportAutoPromotion(branch string) {
	fmt.Printf("🚫 Cannot auto-promote builds of %s branch, please checkout another branch and try again.\n", style.Resource(branch))
}

func (s *InteractivePromptProvider) PrintNotCreatingPullRequest() {
	fmt.Println("👋 Alright, so long my friend!")
}

func (s *InteractivePromptProvider) PrintPromotionAlreadyConfigured(branch, env string) {
	fmt.Printf("🤷 Branch %s pull request is already configured to auto-promote to %s environment.\n", style.Resource(branch), style.Resource(env))
}

func (s *InteractivePromptProvider) PrintPromotionConfigured(branch string, env string) {
	fmt.Printf("✅ Configured auto-promotion of branch %s pull request to %s environment.\n", style.Resource(branch), style.Resource(env))
}

func (s *InteractivePromptProvider) PrintPromotionNotConfigured(branch string, env string) {
	fmt.Printf("🤷 Branch %s pull request was %s configured to auto-promote to %s environment.\n", style.Resource(branch), style.Warning("not"), style.Resource(env))
}

func (s *InteractivePromptProvider) PrintPromotionDisabled(branch string) {
	fmt.Printf("🛑 Disabled auto-promotion of branch %s pull request.\n", style.Resource(branch))
}
