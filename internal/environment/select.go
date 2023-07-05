package environment

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy-cli/internal/config"
	"github.com/nestoca/joy-cli/internal/git"
	"sort"
)

func Select(configFilePath string) error {
	err := git.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return err
	}

	// Load fresh copy of config file, without any alterations/defaults applied
	cfg, err := config.LoadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("loading config file %s: %w", configFilePath, err)
	}

	// Load environments
	envs, err := LoadAll(DirName)
	if err != nil {
		return fmt.Errorf("loading environments: %w", err)
	}

	// Create list of environment names
	var envNames []string
	for _, env := range envs {
		envNames = append(envNames, env.Name)
	}
	sort.Strings(envNames)

	// Prompt user to select environments
	defaultSelected := cfg.Environments.Selected
	if len(defaultSelected) == 0 {
		defaultSelected = envNames
	}
	var selected []string
	err = survey.AskOne(&survey.MultiSelect{
		Message: "Select environments to work with:",
		Options: envNames,
		Default: defaultSelected,
	},
		&selected,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return fmt.Errorf("prompting for environments: %w", err)
	}

	// At least two environments must be selected in order to select a source and target environments.
	// Otherwise, just leave their values unchanged in config.
	if len(selected) >= 2 {
		// Prompt user to select source environment within selected environments
		defaultSource := getDefaultValueWithinOptions(cfg.Environments.Source, selected)
		err = survey.AskOne(&survey.Select{
			Message: "Select source/current environment:",
			Options: selected,
			Default: defaultSource,
		},
			&cfg.Environments.Source,
			survey.WithPageSize(10),
		)
		if err != nil {
			return fmt.Errorf("prompting for source environment: %w", err)
		}

		// Exclude source environment from target environment options
		var targetOptions []string
		for _, env := range selected {
			if env != cfg.Environments.Source {
				targetOptions = append(targetOptions, env)
			}
		}

		// Prompt user to select target environment within selected environments
		defaultTarget := getDefaultValueWithinOptions(cfg.Environments.Target, targetOptions)
		err = survey.AskOne(&survey.Select{
			Message: "Select target/promotion environment:",
			Options: targetOptions,
			Default: defaultTarget,
		},
			&cfg.Environments.Target,
			survey.WithPageSize(10),
		)
		if err != nil {
			return fmt.Errorf("prompting for target environment: %w", err)
		}
	}

	// If all environments are selected, don't explicitly list them in config file,
	// so that new environments are automatically included in selection.
	if len(selected) == len(envNames) {
		selected = nil
	}
	cfg.Environments.Selected = selected

	// Save config
	err = cfg.Save()
	if err != nil {
		return fmt.Errorf("saving config file %s: %w", configFilePath, err)
	}
	fmt.Println("✅ Config updated.")
	return nil
}

// getDefaultValueWithinOptions returns the default value if it is within the given options, otherwise nil.
func getDefaultValueWithinOptions(value string, options []string) interface{} {
	for _, option := range options {
		if option == value {
			return value
		}
	}
	return nil
}
