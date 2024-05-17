package github

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
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
func executeInteractively(workDir string, args ...string) error {
	err := EnsureInstalledAndAuthenticated()
	if err != nil {
		return err
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running gh command with args %q: %w", strings.Join(args, " "), err)
	}
	return nil
}

// ExecuteAndGetOutput runs gh command with given args and returns the stdout output.
func ExecuteAndGetOutput(workDir string, args ...string) (string, error) {
	if err := EnsureInstalledAndAuthenticated(); err != nil {
		return "", err
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running gh command with args %q: %w: %q", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func EnsureInstalledAndAuthenticated() error {
	if err := dependency.MustBeInstalled(); err != nil {
		return err
	}

	cmd := exec.Command("gh", "auth", "status")

	if _, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("🔐 Please run %s to authenticate the gh cli.\n", style.Code("gh auth login"))
		return errors.New("gh cli not authenticated")
	}

	return nil
}

type CloneOptions struct {
	Repo   string
	OutDir string
}

func Clone(dir string, opts CloneOptions) error {
	args := []string{"repo", "clone", opts.Repo}
	if opts.OutDir != "" {
		args = append(args, opts.OutDir)
	}
	_, err := ExecuteAndGetOutput(dir, args...)
	return err
}

func CloneInteractive(dir string, opts CloneOptions) error {
	args := []string{"repo", "clone", opts.Repo}
	if opts.OutDir != "" {
		args = append(args, opts.OutDir)
	}
	return executeInteractively(dir, args...)
}
