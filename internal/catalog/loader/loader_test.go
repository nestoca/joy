package loader

import (
	"context"
	"fmt"
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

func writeEnvironment(t *testing.T, dir, name string) {
	t.Helper()
	content := fmt.Sprintf("apiVersion: joy.nesto.ca/v1alpha1\nkind: Environment\nmetadata:\n  name: %s\n", name)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o644))
}

// newCatalogRepo returns a catalog git repository whose first commit has a dev environment
// and whose second commit adds a prod environment.
func newCatalogRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	dir := filepath.Join(t.TempDir(), "catalog")
	require.NoError(t, os.Mkdir(dir, 0o755))
	runGit(t, dir, "init", "--quiet", "--initial-branch", "master")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")
	for _, env := range []string{"dev", "prod"} {
		writeEnvironment(t, dir, env)
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "--quiet", "--message", env)
	}
	return dir
}

func TestLoad(t *testing.T) {
	dir := newCatalogRepo(t)

	cfg, cat, err := Load(context.Background(), dir)
	require.NoError(t, err)
	require.Equal(t, dir, cfg.CatalogDir)
	require.Equal(t, []string{"dev", "prod"}, cat.GetEnvironmentNames())
}

func TestWorktreeLoaderLoadAt(t *testing.T) {
	dir := newCatalogRepo(t)

	cat, remove, err := NewWorktreeLoader(dir).LoadAt(context.Background(), "master^")
	require.NoError(t, err)
	require.Equal(t, []string{"dev"}, cat.GetEnvironmentNames())

	worktree := filepath.Dir(cat.Environments[0].File.Path)
	require.Equal(t, "catalog", filepath.Base(worktree), "worktree must keep the catalog directory name")
	require.NoError(t, remove())
	require.NoDirExists(t, worktree)
}

func TestWorktreeLoaderLoadAtUnknownRef(t *testing.T) {
	dir := newCatalogRepo(t)

	_, _, err := NewWorktreeLoader(dir).LoadAt(context.Background(), "does-not-exist")
	require.ErrorContains(t, err, "does-not-exist")
}

func TestWorktreeLoaderLoadAtWithRelativeDir(t *testing.T) {
	dir := newCatalogRepo(t)
	t.Chdir(dir)

	cat, remove, err := NewWorktreeLoader(".").LoadAt(context.Background(), "master^")
	require.NoError(t, err)

	worktree := filepath.Dir(cat.Environments[0].File.Path)
	require.Equal(t, "catalog", filepath.Base(worktree), "worktree must keep the catalog directory name")
	require.NoError(t, remove())
	require.NoDirExists(t, worktree)
	require.NoDirExists(t, filepath.Dir(worktree))
}
