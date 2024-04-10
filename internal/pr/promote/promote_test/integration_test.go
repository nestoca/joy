package promote_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/github"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/nestoca/joy/internal/testutils"
)

func TestPromotions(t *testing.T) {
	gitRepo := testutils.CloneToTempDir(t, "joy-pr-promote-test")

	t.Run("testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo", func(t *testing.T) {
		testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo(t, gitRepo)
	})

	t.Run("testGetCurrentBranch", func(t *testing.T) {
		testGetCurrentBranch(t, gitRepo)
	})
}

func testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo(t *testing.T, dir string) {
	branch := "branch-with-pr"
	guid, err := uuid.NewRandom()
	require.NoError(t, err)
	expectedEnv := guid.String()
	require.NoError(t, git.Checkout(dir, branch))
	prProvider := github.NewPullRequestProvider(dir)

	err = prProvider.SetPromotionEnvironment(branch, expectedEnv)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prProvider.SetPromotionEnvironment(branch, "")
		deletePromotionLabelFromRepo(t, dir, expectedEnv)
	})

	actualEnv, err := prProvider.GetPromotionEnvironment(branch)
	require.NoError(t, err)
	require.Equal(t, expectedEnv, actualEnv)
}

func testGetCurrentBranch(t *testing.T, dir string) {
	expectedBranch := "branch-with-pr"
	branchProvider := promote.NewGitBranchProvider(dir)
	require.NoError(t, git.Checkout(dir, expectedBranch))

	actualBranch, err := branchProvider.GetCurrentBranch()

	require.NoError(t, err, "getting current branch")
	require.Equal(t, expectedBranch, actualBranch)
}

func deletePromotionLabelFromRepo(t *testing.T, dir, env string) {
	t.Helper()
	label := fmt.Sprintf("promote:%s", env)
	cmd := exec.Command("gh", "label", "delete", label, "--yes")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "delete label %s from repo", label)
}
