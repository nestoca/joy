package promote

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
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
	fmt.Println("🚫 Cannot promote builds of master/main branch, please create a feature branch and try again.")
}

func (s *SurveyPrompt) PrintNotCreatingPullRequest() {
	fmt.Println("👋 Alright, so long my friend!")
}

func (s *SurveyPrompt) PrintPromotionConfigured(branch string, env string) {
	fmt.Printf("✅ Configured promotion for builds of branch %s PR to %s environment.", branch, env)
}

func (s *SurveyPrompt) PrintPromotionDisabled(branch string) {
	fmt.Printf("🛑 Disabled promotion for builds of branch %s PR.", branch)
}
