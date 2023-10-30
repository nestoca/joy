package promote_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/pr/promote"
)

func TestGetCurrentBranch(t *testing.T) {
	// Prepare
	expectedBranch := "branch-with-pr"
	branchProvider := &promote.GitBranchProvider{}
	assert.NoError(t, git.Checkout(".", expectedBranch))

	// Perform test
	actualBranch, err := branchProvider.GetCurrentBranch()

	// Assert
	assert.NoError(t, err, "getting current branch")
	assert.Equal(t, expectedBranch, actualBranch)
}
