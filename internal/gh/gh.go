package gh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CreatePullRequest creates a pull request.
func CreatePullRequest(args ...string) error {
	err := ensureGHInstalled()
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

func ensureGHInstalled() error {
	cmd := exec.Command("which", "gh")
	err := cmd.Run()
	if err != nil {
		fmt.Println("🤓 This command requires the gh cli.\nSee: https://github.com/cli/cli")
		return errors.New("missing gh cli dependency")
	}
	return nil
}
