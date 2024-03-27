package git

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/style"
)

func EnsureCleanAndUpToDateWorkingCopy(ctx context.Context) error {
	if config.FlagsFromContext(ctx).SkipCatalogUpdate {
		_, _ = fmt.Fprintln(os.Stderr, "ℹ️ Skipping catalog update and dirty check.")
		return nil
	}

	dir := config.FromContext(ctx).CatalogDir

	changes, err := GetUncommittedChanges(dir)
	if err != nil {
		return fmt.Errorf("getting uncommitted changes: %w", err)
	}
	if len(changes) > 0 {
		return fmt.Errorf("uncommitted catalog changes detected:\n%s", style.Warning(strings.Join(changes, "\n")))
	}

	const defaultBranch = "master"
	if err = Checkout(dir, defaultBranch); err != nil {
		return fmt.Errorf("checking out default branch: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "ℹ️ Catalog: checking out %s branch\n", style.Code(defaultBranch))

	if err = Pull(dir); err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}

	return nil
}
