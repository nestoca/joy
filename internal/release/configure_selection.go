package release

import (
	"fmt"
	"sort"

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

	// Select all releases without prompting user?
	if all {
		userCfg.Releases.Selected = nil
		if err := userCfg.Save(); err != nil {
			return fmt.Errorf("saving config file %s: %w", configFilePath, err)
		}
		fmt.Println("✅ Selected all releases.")
		return nil
	}

	// Create list of release names
	var releaseNames []string
	for _, rel := range cat.Releases.Items {
		releaseNames = append(releaseNames, rel.Name)
	}
	sort.Strings(releaseNames)

	// Prompt user to select releases
	defaultSelected := userCfg.Releases.Selected
	if len(defaultSelected) == 0 {
		defaultSelected = releaseNames
	}
	var selected []string
	if err := survey.AskOne(
		&survey.MultiSelect{
			Message: "Select releases to work with:",
			Options: releaseNames,
			Default: defaultSelected,
		},
		&selected,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required),
	); err != nil {
		return fmt.Errorf("prompting for releases: %w", err)
	}

	// If all releases are selected, don't explicitly list them in config file,
	// so that new releases are automatically included in selection.
	if len(selected) == len(releaseNames) {
		selected = nil
	}
	userCfg.Releases.Selected = selected

	if err := userCfg.Save(); err != nil {
		return fmt.Errorf("saving config file %s: %w", configFilePath, err)
	}

	fmt.Println("✅ Config updated.")
	return nil
}
