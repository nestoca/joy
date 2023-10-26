package github

import (
	"errors"
	"fmt"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var dependency = &dependencies.Dependency{
	Command:    "gh",
	Url:        "https://github.com/cli/cli",
	IsRequired: true,
}

func init() {
	dependencies.Add(dependency)
}

// executeInteractively runs gh command with given args with full forwarding of stdin, stdout and stderr.
func executeInteractively(args ...string) error {
	err := EnsureInstalledAndAuthenticated()
	if err != nil {
		return err
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("running gh command with args %q: %w", strings.Join(args, " "), err)
	}
	return nil
}

// executeAndGetOutput runs gh command with given args and returns the stdout output.
func executeAndGetOutput(args ...string) (string, error) {
	err := EnsureInstalledAndAuthenticated()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running gh command with args %q: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

func EnsureInstalledAndAuthenticated() error {
	dependency.MustBeInstalled()

	// Check if user is logged in
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		fmt.Printf("🔐 Please run %s to authenticate the gh cli.\n", style.Code("gh auth login"))
		return errors.New("gh cli not authenticated")
	}

	// Check if user has required scopes
	scopesRegex := regexp.MustCompile(`Token scopes: .*?\b(read:org)\b.*?\b(repo)\b.*`)
	if !scopesRegex.MatchString(outputStr) {
		fmt.Printf("🔐 Please ensure you have the following permission scopes: %s, %s\n", style.Code("read:org"), style.Code("repo"))
		return errors.New("gh cli token missing required scopes")
	}

	return nil
}
