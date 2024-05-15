package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/nestoca/joy/internal/style"
)

func EnsureCleanAndUpToDateWorkingCopy(catalogDir string, out io.Writer) error {
	changes, err := GetUncommittedChanges(catalogDir)
	if err != nil {
		return fmt.Errorf("getting uncommitted changes: %w", err)
	}
	if len(changes) > 0 {
		return fmt.Errorf("uncommitted catalog changes detected:\n%s", style.Warning(strings.Join(changes, "\n")))
	}

	const (
		defaultBranch = "master"
		originBranch  = "origin/" + defaultBranch
	)

	if err := Checkout(catalogDir, defaultBranch); err != nil {
		return fmt.Errorf("checking out default branch: %w", err)
	}
	_, _ = fmt.Fprintf(out, "ℹ️ Catalog: checking out %s branch\n", style.Code(defaultBranch))

	if err := Pull(catalogDir); err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}

	diffs, err := Diff(catalogDir, originBranch)
	if err != nil {
		return fmt.Errorf("running diff against origin: %w", err)
	}

	if len(diffs) > 0 {
		return fmt.Errorf("diff with %s detected:\n%s", style.Code(originBranch), style.Warning(strings.Join(diffs, "\n")))
	}

	return nil
}
