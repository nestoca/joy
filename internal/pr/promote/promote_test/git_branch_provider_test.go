package promote_test

import (
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	// Prepare
	expectedBranch := "branch-with-pr"
	branchProvider := &promote.GitBranchProvider{}
	checkOut(t, expectedBranch)

	// Perform test
	actualBranch, err := branchProvider.GetCurrentBranch()

	// Assert
	assert.NoError(t, err, "getting current branch")
	assert.Equal(t, expectedBranch, actualBranch)
}
