package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func newRepoWithTwoCommits(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet", "--initial-branch", "master")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")
	for _, content := range []string{"first", "second"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte(content), 0o644))
		runGit(t, dir, "add", "file.txt")
		runGit(t, dir, "commit", "--quiet", "--message", content)
	}
	return dir
}

func TestCheckoutWorktree(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := newRepoWithTwoCommits(t)

	path, remove, err := CheckoutWorktree(repo, "test/catalog", "master^")
	require.NoError(t, err)
	require.Equal(t, "catalog", filepath.Base(path))

	content, err := os.ReadFile(filepath.Join(path, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "first", string(content))

	require.NoError(t, remove())
	require.NoDirExists(t, path)
}

func TestCheckoutWorktreeAtCheckedOutBranch(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := newRepoWithTwoCommits(t)

	path, remove, err := CheckoutWorktree(repo, "test", "master")
	require.NoError(t, err)
	defer func() { require.NoError(t, remove()) }()

	content, err := os.ReadFile(filepath.Join(path, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "second", string(content))
}

func TestCheckoutWorktreeAtUnknownRef(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := newRepoWithTwoCommits(t)

	_, _, err := CheckoutWorktree(repo, "test", "does-not-exist")
	require.ErrorContains(t, err, "resolving ref does-not-exist")
}
