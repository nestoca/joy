package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/style"
)

func EnsureCleanAndUpToDateWorkingCopy(ctx context.Context) error {
	if config.FlagsFromContext(ctx).SkipCatalogUpdate {
		fmt.Println("ℹ️ Skipping catalog update and dirty check.")
		return nil
	}

	dir := config.FromContext(ctx).CatalogDir

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

	if err = Checkout(dir, defaultBranch); err != nil {
		return fmt.Errorf("checking out default branch: %w", err)
	}
	fmt.Printf("ℹ️ Catalog: checking out %s branch\n", defaultBranch)

	if err = Pull(dir); err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}

	return nil
}
