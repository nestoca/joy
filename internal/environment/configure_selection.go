package environment

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/pkg/catalog"
)

func ConfigureSelection(catalogDir, configFilePath string, all bool) error {
	err := git.EnsureCleanAndUpToDateWorkingCopy(catalogDir)
	if err != nil {
		return err
	}

	// Load fresh copy of config file, without any alterations/defaults applied
	cfg, err := config.LoadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("loading config file %s: %w", configFilePath, err)
	}

	// Select all environments without prompting user?
	if all {
		cfg.Environments.Selected = nil
		err = cfg.Save()
		if err != nil {
			return fmt.Errorf("saving config file %s: %w", configFilePath, err)
		}
		fmt.Println("✅ Selected all environments.")
		return nil
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		Dir:             catalogDir,
		LoadEnvs:        true,
		SortEnvsByOrder: true,
	}
	cat, err := catalog.Load(loadOpts)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Create list of environment names
	var envNames []string
	for _, env := range cat.Environments {
		envNames = append(envNames, env.Name)
	}

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
