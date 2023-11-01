package promote

import (
	"fmt"

	"github.com/nestoca/joy/internal/git"
)

type ShellGitProvider struct {
	dir string
}

func NewShellGitProvider(dir string) *ShellGitProvider {
	return &ShellGitProvider{dir: dir}
}

func (g *ShellGitProvider) EnsureCleanAndUpToDateWorkingCopy() error {
	return git.EnsureCleanAndUpToDateWorkingCopy(g.dir)
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

func (g *ShellGitProvider) CheckoutMasterBranch() error {
	err := git.Checkout(g.dir, "master")
	if err != nil {
		return fmt.Errorf("checking out master branch: %w", err)
	}
	return nil
}
