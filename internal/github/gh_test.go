package github

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFetchAuthenticatesWithoutGitCredentialHelper reproduces CI, where git has no credential helper configured
// and only gh is authenticated (eg: through GH_TOKEN), so plain git cannot fetch private repositories.
func TestFetchAuthenticatesWithoutGitCredentialHelper(t *testing.T) {
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh is not authenticated")
	}

	dir := t.TempDir()
	require.NoError(t, Clone(dir, CloneOptions{Repo: "nestoca/joy-release-promote-test", OutDir: "repo"}))

	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repoDir := filepath.Join(dir, "repo")
	require.NoError(t, Fetch(repoDir))
	require.NoError(t, Fetch(repoDir, "--tags"))
}
