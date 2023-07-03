package promote

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release"
	"github.com/nestoca/joy-cli/internal/release/cross"
	"path/filepath"
)

const darkGrey = "\033[38;2;90;90;90m"
const darkYellow = "\033[38;2;128;128;0m"

type Opts struct {
	// BaseDir is the catalog base directory.
	BaseDir string

	// SourceEnv is the source environment.
	SourceEnv string

	// TargetEnv is the target environment.
	TargetEnv string

	// Action to perform.
	// Optional, defaults to prompting user.
	Action string

	// Filter specifies releases to promote.
	// Optional, defaults to prompting user.
	Filter release.Filter
}

const (
	ActionPreview       = "Preview"
	ActionApply         = "Apply"
	ActionApplyNoCommit = "Apply (but don't commit)"
	ActionCancel        = "Cancel"
)

func Promote(opts Opts) error {
	environmentsDir := filepath.Join(opts.BaseDir, "environments")
	environments := environment.NewList([]string{opts.SourceEnv, opts.TargetEnv})
	list, err := cross.Load(environmentsDir, environments, opts.Filter)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	if opts.Filter == nil {
		list, err = SelectReleases(opts.SourceEnv, opts.TargetEnv, list)
		if err != nil {
			return fmt.Errorf("selecting releases to promote: %w", err)
		}
	}

	if len(list.Releases) == 0 {
		fmt.Println("No releases to promote.")
		return nil
	} else if opts.Filter != nil {
		// Print releases that were matched by filter.
		list.Print()
	}

	for {
		// Determine action to perform.
		action := opts.Action
		interactive := false
		if opts.Action == "" {
			interactive = true
			action, err = SelectAction()
			if err != nil {
				return fmt.Errorf("selecting action: %w", err)
			}
		}

		switch action {
		case ActionPreview:
			err := Preview(list)
			if err != nil {
				return fmt.Errorf("previewing: %w", err)
			}
			if !interactive {
				return nil
			}
		case ActionApply:
			err = Apply(list, true)
			if err != nil {
				return fmt.Errorf("applying: %w", err)
			}
			return nil
		case ActionApplyNoCommit:
			err = Apply(list, false)
			if err != nil {
				return fmt.Errorf("applying (no commit): %w", err)
			}
			return nil
		case ActionCancel:
			return nil
		}
	}
}

func Apply(list *cross.ReleaseList, commit bool) error {
	// TODO
	return nil
}

// SelectAction prompts user for action to perform.
func SelectAction() (string, error) {
	actions := []string{ActionPreview, ActionApply, ActionApplyNoCommit, ActionCancel}
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
