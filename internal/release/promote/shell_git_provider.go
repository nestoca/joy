package promote

import (
	"fmt"

	"github.com/nestoca/joy/internal/git"
)

type ShellGitProvider struct {
	dir string
	ref string
}

// NewShellGitProvider returns a GitProvider operating on the catalog in dir, returning to ref once done.
func NewShellGitProvider(dir string, ref string) *ShellGitProvider {
	return &ShellGitProvider{dir: dir, ref: ref}
}

func (g *ShellGitProvider) CreateAndPushBranchWithFiles(branchName string, files []string, message string) error {
	err := git.CreateBranch(g.dir, branchName)
	if err != nil {
		return fmt.Errorf("creating branch %s: %w", branchName, err)
	}

	err = git.Add(g.dir, files)
	if err != nil {
		return fmt.Errorf("adding files to index: %w", err)
	}
	err = git.Commit(g.dir, message)
	if err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}

	err = git.PushNewBranch(g.dir, branchName)
	if err != nil {
		return fmt.Errorf("pushing changes: %w", err)
	}
	return nil
}

func (g *ShellGitProvider) ResetAndForcePushBranchWithFiles(branchName string, expectedHead string, files []string, message string) error {
	if err := git.ResetBranch(g.dir, branchName); err != nil {
		return err
	}
	if err := git.Add(g.dir, files); err != nil {
		return fmt.Errorf("adding files to index: %w", err)
	}
	if err := git.Commit(g.dir, message); err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}
	if err := git.ForcePushBranch(g.dir, branchName, expectedHead); err != nil {
		return fmt.Errorf("pushing changes: %w", err)
	}
	return nil
}

func (g *ShellGitProvider) DiscardChanges() error {
	return git.DiscardChanges(g.dir)
}

func (g *ShellGitProvider) CheckoutCatalogRef() error {
	return git.Checkout(g.dir, g.ref)
}
