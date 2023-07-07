package promote

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/release"
)

type Opts struct {
	// SourceEnv is the source environment.
	SourceEnv string

	// TargetEnv is the target environment.
	TargetEnv string

	// Filter specifies releases to promote.
	// Optional, defaults to prompting user.
	Filter release.Filter
}

// Promote prepares a promotion, shows a preview to user and asks for confirmation before performing it.
func Promote(opts Opts) error {
	err := git.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return err
	}

	// Load matching releases from given environments.
	environments, err := environment.LoadAll(environment.DirName, opts.SourceEnv, opts.TargetEnv)
	if err != nil {
		return fmt.Errorf("loading environments: %w", err)
	}
	list, err := release.LoadCrossReleaseList(environment.DirName, environments, opts.Filter)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	// Keep only promotable releases.
	list = list.OnlyPromotableReleases()
	if len(list.Releases) == 0 {
		fmt.Println("🤷 No promotable releases found.")
		return nil
	}

	err = CreateMissingTargetReleases(list)
	if err != nil {
		return fmt.Errorf("creating missing releases: %w", err)
	}

	// Count matching releases.
	releaseCount := len(list.Releases)
	if releaseCount == 0 {
		if opts.Filter != nil {
			fmt.Println("🤷 Given filter matched no releases.")
		} else {
			fmt.Println("🤷 Given environments contain no releases.")
		}
		return nil
	}

	// Promote user to select releases
	list, err = SelectReleases(opts.SourceEnv, opts.TargetEnv, list)
	if err != nil {
		return fmt.Errorf("selecting releases to promote: %w", err)
	}

	// Print selected releases
	fmt.Println(MajorSeparator)
	list.Print(release.PrintOpts{IsPromoting: true})

	// Show preview.
	err = preview(list)
	if err != nil {
		return fmt.Errorf("previewing: %w", err)
	}

	// Ask for confirmation before performing promotion.
	actions := []string{"Create PR", "Cancel"}
	prompt := &survey.Select{
		Message: "What do you want to do?",
		Options: actions,
	}
	var selectedAction string
	err = survey.AskOne(prompt, &selectedAction)
	if err != nil {
		return fmt.Errorf("asking user for confirmation: %w", err)
	}

	// Cancel?
	if selectedAction == "Cancel" {
		fmt.Println("👋 OK, so long my friend!")
		return nil
	}

	// Perform promotion.
	err = perform(list)
	if err != nil {
		return fmt.Errorf("applying: %w", err)
	}
	return nil
}
