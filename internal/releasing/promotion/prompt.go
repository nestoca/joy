package promotion

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/colors"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/git"
	"github.com/nestoca/joy-cli/internal/releasing"
)

type Opts struct {
	// BaseDir is the catalog base directory.
	BaseDir string

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
	Filter releasing.Filter
}

const (
	ActionPreview = "Preview"
	ActionPromote = "Promote"
	ActionCancel  = "Cancel"
)

// Prompt prompts user for different selections and actions to perform,
// such as previewing or promoting releases.
func Prompt(opts Opts) error {
	err := git.EnsureNoUncommittedChanges()
	if err != nil {
		return err
	}

	// Load matching releases from given environments.
	environmentsDir := "environments"
	environments := environment.NewList([]string{opts.SourceEnv, opts.TargetEnv})
	list, err := releasing.LoadCrossReleaseList(environmentsDir, environments, opts.Filter)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	// Count matching releases.
	releaseCount := len(list.Releases)
	if releaseCount == 0 {
		if opts.Filter != nil {
			fmt.Println("🤷Given filter matched no releases.")
		} else {
			fmt.Println("🤷Given environments contain no releases.")
		}
		return nil
	}

	// Prompt user to select releases?
	if opts.Filter == nil {
		list, err = SelectReleases(opts.SourceEnv, opts.TargetEnv, list)
		if err != nil {
			return fmt.Errorf("selecting releases to promote: %w", err)
		}

		// Count releases selected by user.
		releaseCount = len(list.Releases)
		if releaseCount == 0 {
			fmt.Println("🤷No releases selected.")
			return nil
		}
	}

	// Print releases that were matched by filter.
	plural := ""
	if releaseCount > 1 {
		plural = "s"
	}
	nonPromotableCount := 0
	for _, release := range list.Releases {
		if !release.Promotable() {
			nonPromotableCount++
		}
	}
	nonPromotable := ""
	if nonPromotableCount > 0 {
		plural := "is"
		if nonPromotableCount > 1 {
			plural = "are"
		}
		nonPromotable = fmt.Sprintf(" (%d of which %s not promotable)", nonPromotableCount, plural)
	}
	fmt.Println(MajorSeparator)
	fmt.Printf("🔍%s %d release%s%s\n", color.InWhite("Matching"), releaseCount, plural, colors.InDarkGrey(nonPromotable))
	fmt.Println(MinorSeparator)
	list.Print(releasing.PrintOpts{IsPromoting: true})

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
