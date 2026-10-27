package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/nestoca/joy/internal/style"
)

// EnsureCleanAndUpToDateWorkingCopy checks out the given ref in the catalog after ensuring there are no local changes.
// When the ref is a remote branch, it is also pulled and verified to match its remote counterpart.
func EnsureCleanAndUpToDateWorkingCopy(catalogDir string, ref string, out io.Writer) error {
	changes, err := GetUncommittedChanges(catalogDir)
	if err != nil {
		return fmt.Errorf("getting uncommitted changes: %w", err)
	}
	if len(changes) > 0 {
		return fmt.Errorf("uncommitted catalog changes detected:\n%s", style.Warning(strings.Join(changes, "\n")))
	}

	if err := Fetch(catalogDir); err != nil {
		return fmt.Errorf("fetching changes: %w", err)
	}

	isBranch, err := IsRemoteBranch(catalogDir, ref)
	if err != nil {
		return fmt.Errorf("checking if %s is a remote branch: %w", ref, err)
	}

	if !isBranch {
		return checkoutDetachedRef(catalogDir, ref, out)
	}

	if err := Checkout(catalogDir, ref); err != nil {
		return fmt.Errorf("checking out %s: %w", ref, err)
	}
	_, _ = fmt.Fprintf(out, "ℹ️  Catalog: checking out %s\n", style.Code(ref))

	if err := Pull(catalogDir); err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}

	originRef := "origin/" + ref
	diffs, err := Diff(catalogDir, originRef)
	if err != nil {
		return fmt.Errorf("running diff against origin: %w", err)
	}

	if len(diffs) > 0 {
		return fmt.Errorf("diff with %s detected:\n%s", style.Code(originRef), style.Warning(strings.Join(diffs, "\n")))
	}

	return nil
}

// checkoutDetachedRef checks out the commit that ref resolves to and verifies that HEAD now points to it.
func checkoutDetachedRef(catalogDir string, ref string, out io.Writer) error {
	commit, err := ResolveCommit(catalogDir, ref)
	if err != nil {
		return err
	}

	if err := Checkout(catalogDir, commit); err != nil {
		return fmt.Errorf("checking out %s: %w", ref, err)
	}
	_, _ = fmt.Fprintf(out, "ℹ️  Catalog: checking out %s (%s)\n", style.Code(ref), commit)

	head, err := GetCurrentCommit(catalogDir)
	if err != nil {
		return err
	}
	if head != commit {
		return fmt.Errorf("checked out %s but HEAD is at %s instead of %s", ref, head, commit)
	}

	return nil
}
