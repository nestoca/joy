package promote

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/style"
)

type SurveyPrompt struct {
}

func (s *SurveyPrompt) WhetherToCreateMissingPullRequest() (bool, error) {
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

func (s *SurveyPrompt) WhichEnvironmentToPromoteTo(environments []string, preSelectedEnv string) (string, error) {
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

func (s *SurveyPrompt) PrintMasterBranchPromotion() {
	fmt.Printf("🚫 Cannot promote builds of %s/%s branch, please create a feature branch and try again.\n", style.Resource("master"), style.Resource("main"))
}

func (s *SurveyPrompt) PrintNotCreatingPullRequest() {
	fmt.Println("👋 Alright, so long my friend!")
}

func (s *SurveyPrompt) PrintPromotionConfigured(branch string, env string) {
	fmt.Printf("✅ Configured promotion of branch %s pull request to %s environment.\n", style.Resource(branch), style.Resource(env))
}

func (s *SurveyPrompt) PrintPromotionDisabled(branch string) {
	fmt.Printf("🛑 Disabled promotion of branch %s pull request.\n", style.Resource(branch))
}
