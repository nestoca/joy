package git

import (
	"fmt"
	"strings"

	"github.com/nestoca/joy/internal/style"
)

func EnsureCleanAndUpToDateWorkingCopy(dir string) error {
	changes, err := GetUncommittedChanges(dir)
	if err != nil {
		return fmt.Errorf("getting uncommitted changes: %w", err)
	}
	if len(changes) > 0 {
		return fmt.Errorf("uncommitted changes detected:\n%s", style.Warning(strings.Join(changes, "\n")))
	}

	defaultBranch, err := GetDefaultBranch(dir)
	if err != nil {
		return fmt.Errorf("getting default branch: %w", err)
	}
	err = Checkout(dir, defaultBranch)
	if err != nil {
		return fmt.Errorf("checking out default branch: %w", err)
	}
	err = Pull(dir)
	if err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}

	return nil
}
