package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// CloneToTempDir will clone given nestoca repo to a temporary directory and return its absolute path.
func CloneToTempDir(t *testing.T, repoName string) string {
	tempDir, err := os.MkdirTemp("", repoName+"-")
	require.NoError(t, err)

	repoURL := func() string {
		if gitToken := os.Getenv("GH_TOKEN"); gitToken != "" {
			return fmt.Sprintf("https://%s@github.com/nestoca/%s.git", gitToken, repoName)
		}
		return fmt.Sprintf("git@github.com:nestoca/%s.git", repoName)
	}()

	require.NoError(t, cmd("git", "clone", repoURL, tempDir).Run())

	return tempDir
}

func cmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}
