package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func RunTestsInClonedRepo(m *testing.M, repoName string) {
	exitCode := 0
	defer os.Exit(exitCode)

	// Clone test repo in temp directory
	cloneDir, err := os.MkdirTemp("", repoName+"-")
	if err != nil {
		panic(fmt.Errorf("creating temp dir: %w", err))
	}
	oldWorkDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("getting current working dir: %w", err))
	}
	err = os.Chdir(cloneDir)
	if err != nil {
		panic(fmt.Errorf("changing to temp dir: %w", err))
	}
	repoUrl := fmt.Sprintf("git@github.com:nestoca/%s.git", repoName)
	err = exec.Command("git", "clone", repoUrl, ".").Run()
	if err != nil {
		panic(fmt.Errorf("cloning test repo: %w", err))
	}
	defer func() {
		_ = os.Chdir(oldWorkDir)
		_ = os.RemoveAll(cloneDir)
	}()

	exitCode = m.Run()
}
