package promote_test

import (
	"testing"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentBranch(t *testing.T) {
	dir := "."
	// Prepare
	expectedBranch := "branch-with-pr"
	branchProvider := promote.NewGitBranchProvider(dir)
	assert.NoError(t, git.Checkout(dir, expectedBranch))

	// Perform test
	actualBranch, err := branchProvider.GetCurrentBranch()

	// Assert
	assert.NoError(t, err, "getting current branch")
	assert.Equal(t, expectedBranch, actualBranch)
}
