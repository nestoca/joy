package promote

import (
	"fmt"
	"os/exec"
	"strings"
)

type GitBranchProvider struct {
	dir string
}

func NewGitBranchProvider(dir string) *GitBranchProvider {
	return &GitBranchProvider{
		dir: dir,
	}
}

func (g *GitBranchProvider) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "-C", g.dir, "symbolic-ref", "--short", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return "", fmt.Errorf("getting name of current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
