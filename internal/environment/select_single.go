package environment

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/api/v1alpha1"
)

func SelectSingle(environments []*v1alpha1.Environment, current *v1alpha1.Environment, message string) (*v1alpha1.Environment, error) {
	if len(environments) == 0 {
		return nil, fmt.Errorf("no environments found")
	}

	// Create list of environment names
	var envNames []string
	for _, env := range environments {
		envNames = append(envNames, env.Name)
	}

	// Find index of current environment
	var selectedIndex int
	for i, env := range environments {
		if env == current {
			selectedIndex = i
			break
		}
	}

	// Prompt user to select environment
	err := survey.AskOne(&survey.Select{
		Message: message,
		Options: envNames,
		Default: selectedIndex,
	},
		&selectedIndex,
		survey.WithPageSize(10),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for environment: %w", err)
	}
	return environments[selectedIndex], nil
}
