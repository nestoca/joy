package release

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/git"
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

	// Load all releases across all environments
	environments, err := environment.LoadAll(environment.DirName)
	if err != nil {
		return fmt.Errorf("loading environments: %w", err)
	}
	list, err := LoadCrossReleaseList(environment.DirName, environments, nil)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	// Create list of release names
	var releaseNames []string
	for _, rel := range list.Releases {
		releaseNames = append(releaseNames, rel.Name)
	}
	sort.Strings(releaseNames)

	// Prompt user to select releases
	defaultSelected := cfg.Releases.Selected
	if len(defaultSelected) == 0 {
		defaultSelected = releaseNames
	}
	var selected []string
	err = survey.AskOne(&survey.MultiSelect{
		Message: "Select releases to work with:",
		Options: releaseNames,
		Default: defaultSelected,
	},
		&selected,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return fmt.Errorf("prompting for releases: %w", err)
	}

	// If all releases are selected, don't explicitly list them in config file,
	// so that new releases are automatically included in selection.
	if len(selected) == len(releaseNames) {
		selected = nil
	}
	cfg.Releases.Selected = selected

	// Save config
	err = cfg.Save()
	if err != nil {
		return fmt.Errorf("saving config file %s: %w", configFilePath, err)
	}
	fmt.Println("✅ Config updated.")
	return nil
}
