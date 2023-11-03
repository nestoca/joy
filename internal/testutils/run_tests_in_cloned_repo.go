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

	repoUrl := fmt.Sprintf("git@github.com:nestoca/%s.git", repoName)
	require.NoError(t, exec.Command("git", "clone", repoUrl, tempDir).Run())

	return tempDir
}
