package promotion

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/git"
	"github.com/nestoca/joy-cli/internal/release"
)

type Opts struct {
	// SourceEnv is the source environment.
	SourceEnv string

	// TargetEnv is the target environment.
	TargetEnv string

	// Action is the operation to perform.
	// Optional, defaults to prompting user.
	Action string

	// Whether to push changes after committing them.
	Push bool

	// Filter specifies releases to promote.
	// Optional, defaults to prompting user.
	Filter release.Filter
}

const (
	ActionPreview = "Preview"
	ActionPromote = "Promote"
	ActionCancel  = "Cancel"
)

// Prompt prompts user for different selections and actions to perform,
// such as previewing or promoting releases.
func Prompt(opts Opts) error {
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

	// Prompt user to select releases
	list, err = SelectReleases(opts.SourceEnv, opts.TargetEnv, list)
	if err != nil {
		return fmt.Errorf("selecting releases to promote: %w", err)
	}

	// Print selected releases
	fmt.Println(MajorSeparator)
	list.Print(release.PrintOpts{IsPromoting: true})
	fmt.Println(MajorSeparator)

	for {
		// Determine action to perform.
		action := opts.Action
		interactive := false
		if opts.Action == "" {
			interactive = true
			action, err = selectAction()
			if err != nil {
				return fmt.Errorf("selecting action: %w", err)
			}
		}

		switch action {
		case ActionPreview:
			err := preview(list)
			if err != nil {
				return fmt.Errorf("previewing: %w", err)
			}
			if !interactive {
				return nil
			}
		case ActionPromote:
			err = promote(list, opts.Push)
			if err != nil {
				return fmt.Errorf("applying: %w", err)
			}
			return nil
		case ActionCancel:
			return nil
		}
	}
}

// selectAction prompts user for action to perform.
func selectAction() (string, error) {
	actions := []string{ActionPreview, ActionPromote, ActionCancel}
	prompt := &survey.Select{
		Message: "What do you want to do?",
		Options: actions,
	}
	var selectedAction string
	err := survey.AskOne(prompt, &selectedAction)
	if err != nil {
		return "", err
	}
	return selectedAction, nil
}
