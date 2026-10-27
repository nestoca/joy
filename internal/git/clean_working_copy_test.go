package git

import (
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureCleanAndUpToDateWorkingCopy(t *testing.T) {
	cases := []struct {
		Name string
		Ref  func(t *testing.T, dir string) string
	}{
		{
			Name: "branch",
			Ref:  func(t *testing.T, dir string) string { return "master" },
		},
		{
			Name: "commit sha",
			Ref:  func(t *testing.T, dir string) string { return runGit(t, dir, "rev-parse", "HEAD~1") },
		},
		{
			Name: "tag",
			Ref:  func(t *testing.T, dir string) string { return "v1" },
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			dir := setupCatalogClone(t)
			ref := tc.Ref(t, dir)

			require.NoError(t, EnsureCleanAndUpToDateWorkingCopy(dir, ref, io.Discard))
			require.Equal(t, runGit(t, dir, "rev-parse", ref+"^{commit}"), runGit(t, dir, "rev-parse", "HEAD"))
		})
	}
}

func TestEnsureCleanAndUpToDateWorkingCopyUnknownRef(t *testing.T) {
	dir := setupCatalogClone(t)
	before := runGit(t, dir, "rev-parse", "HEAD")

	require.Error(t, EnsureCleanAndUpToDateWorkingCopy(dir, "does-not-exist", io.Discard))
	require.Equal(t, before, runGit(t, dir, "rev-parse", "HEAD"))
}

func TestResolveCommit(t *testing.T) {
	dir := setupCatalogClone(t)

	commit, err := ResolveCommit(dir, "v1")
	require.NoError(t, err)
	require.Equal(t, runGit(t, dir, "rev-parse", "HEAD~1"), commit)

	_, err = ResolveCommit(dir, "does-not-exist")
	require.Error(t, err)
}

func TestIsRemoteBranch(t *testing.T) {
	dir := setupCatalogClone(t)

	isBranch, err := IsRemoteBranch(dir, "master")
	require.NoError(t, err)
	require.True(t, isBranch)

	isBranch, err = IsRemoteBranch(dir, "v1")
	require.NoError(t, err)
	require.False(t, isBranch)
}

// setupCatalogClone creates an origin repository with two commits on master and a v1 tag on the first, and returns the path of a clone of it.
func setupCatalogClone(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	clone := filepath.Join(root, "clone")

	runGit(t, root, "init", "--initial-branch=master", origin)
	runGit(t, origin, "commit", "--allow-empty", "-m", "first")
	runGit(t, origin, "tag", "v1")
	runGit(t, origin, "commit", "--allow-empty", "-m", "second")
	runGit(t, root, "clone", origin, clone)

	return clone
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=joy", "-c", "user.email=joy@example.com", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

func TestIsBranchInSyncWithRemote(t *testing.T) {
	cases := []struct {
		Name     string
		Setup    func(t *testing.T, clone, origin string)
		Expected bool
	}{
		{
			Name:     "in sync",
			Setup:    func(t *testing.T, clone, origin string) {},
			Expected: true,
		},
		{
			Name: "in sync while another branch is checked out",
			Setup: func(t *testing.T, clone, origin string) {
				runGit(t, clone, "checkout", "-b", "feature")
				runGit(t, clone, "commit", "--allow-empty", "-m", "feature")
			},
			Expected: true,
		},
		{
			Name: "behind while another branch is checked out",
			Setup: func(t *testing.T, clone, origin string) {
				runGit(t, origin, "commit", "--allow-empty", "-m", "third")
				runGit(t, clone, "checkout", "--detach", "HEAD")
			},
			Expected: false,
		},
		{
			Name: "ahead",
			Setup: func(t *testing.T, clone, origin string) {
				runGit(t, clone, "commit", "--allow-empty", "-m", "local")
			},
			Expected: false,
		},
		{
			Name: "diverged",
			Setup: func(t *testing.T, clone, origin string) {
				runGit(t, origin, "commit", "--allow-empty", "-m", "remote")
				runGit(t, clone, "commit", "--allow-empty", "-m", "local")
			},
			Expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			clone := setupCatalogClone(t)
			tc.Setup(t, clone, filepath.Join(filepath.Dir(clone), "origin"))

			inSync, err := IsBranchInSyncWithRemote(clone, "master")
			require.NoError(t, err)
			require.Equal(t, tc.Expected, inSync)
		})
	}
}

func TestIsBranchInSyncWithRemoteWithoutLocalBranch(t *testing.T) {
	clone := setupCatalogClone(t)
	origin := filepath.Join(filepath.Dir(clone), "origin")
	runGit(t, origin, "branch", "release")

	inSync, err := IsBranchInSyncWithRemote(clone, "release")
	require.NoError(t, err)
	require.False(t, inSync)
}
