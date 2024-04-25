package environment

import (
	"fmt"

	"github.com/nestoca/survey/v2"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/pkg/catalog"
)

func ConfigureSelection(cat *catalog.Catalog, configFilePath string, all bool) error {
	// Load fresh copy of config file, without any alterations/defaults applied
	userCfg := config.User{FilePath: configFilePath}
	if err := config.LoadFile(userCfg.FilePath, &userCfg); err != nil {
		return fmt.Errorf("loading config file %s: %w", configFilePath, err)
	}

	// Select all environments without prompting user?
	if all {
		userCfg.Environments.Selected = nil
		if err := userCfg.Save(); err != nil {
			return fmt.Errorf("saving config file %s: %w", configFilePath, err)
		}

		fmt.Println("✅ Selected all environments.")
		return nil
	}

	// Create list of environment names
	var envNames []string
	for _, env := range cat.Environments {
		envNames = append(envNames, env.Name)
	}

	// Prompt user to select environments
	defaultSelected := userCfg.Environments.Selected
	if len(defaultSelected) == 0 {
		defaultSelected = envNames
	}
	var selected []string
	if err := survey.AskOne(
		&survey.MultiSelect{
			Message: "Select environments to work with:",
			Options: envNames,
			Default: defaultSelected,
		},
		&selected,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required),
	); err != nil {
		return fmt.Errorf("prompting for environments: %w", err)
	}

	// If all environments are selected, don't explicitly list them in config file,
	// so that new environments are automatically included in selection.
	if len(selected) == len(envNames) {
		selected = nil
	}
	userCfg.Environments.Selected = selected

	if err := userCfg.Save(); err != nil {
		return fmt.Errorf("saving config file %s: %w", configFilePath, err)
	}

	fmt.Println("✅ Config updated.")
	return nil
}
