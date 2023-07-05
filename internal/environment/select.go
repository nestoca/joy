package environment

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy-cli/internal/config"
	"sort"
)

func Select(configFilePath string) error {
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
		survey.WithPageSize(5),
		survey.WithKeepFilter(true),
		survey.WithRemoveSelectNone(),
	)
	if err != nil {
		return fmt.Errorf("prompting for environments: %w", err)
	}

	// Prompt user to select source environment within selected environments
	var source string
	defaultSource := getDefaultValueWithinOptions(cfg.Environments.Source, selected)
	err = survey.AskOne(&survey.Select{
		Message: "Select source/current environment:",
		Options: selected,
		Default: defaultSource,
	},
		&source,
		survey.WithPageSize(5),
	)
	if err != nil {
		return fmt.Errorf("prompting for source environment: %w", err)
	}

	// Exclude source environment from target environment options
	var targetOptions []string
	for _, env := range selected {
		if env != source {
			targetOptions = append(targetOptions, env)
		}
	}

	// Prompt user to select target environment within selected environments
	var target string
	defaultTarget := getDefaultValueWithinOptions(cfg.Environments.Target, targetOptions)
	err = survey.AskOne(&survey.Select{
		Message: "Select target/promotion environment:",
		Options: targetOptions,
		Default: defaultTarget,
	},
		&target,
		survey.WithPageSize(5),
	)
	if err != nil {
		return fmt.Errorf("prompting for target environment: %w", err)
	}

	// If all environments are selected, don't save selected environments to config file.
	// That allows to automatically include new environments in selection.
	if len(selected) == len(envNames) {
		selected = nil
	}

	// Save selected environments to config file
	cfg.Environments.Selected = selected
	cfg.Environments.Source = source
	cfg.Environments.Target = target
	err = cfg.Save()
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
