package promote_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	exitCode := 0
	defer os.Exit(exitCode)

	// Clone test repo in temp directory
	cloneDir, err := os.MkdirTemp("", "joy-pr-promote-test-")
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
	err = exec.Command("git", "clone", "git@github.com:nestoca/joy-pr-promote-test.git", ".").Run()
	if err != nil {
		panic(fmt.Errorf("cloning test repo: %w", err))
	}
	defer func() {
		_ = os.Chdir(oldWorkDir)
		_ = os.RemoveAll(cloneDir)
	}()

	exitCode = m.Run()
}

func checkOut(t *testing.T, branch string) {
	t.Helper()
	cmd := exec.Command("git", "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "checking out branch %s", branch)
}
