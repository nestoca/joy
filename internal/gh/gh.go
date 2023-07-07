package gh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CreatePullRequest creates a pull request.
func CreatePullRequest(args ...string) error {
	cmd := exec.Command("gh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running gh command with args %q: %w", strings.Join(args, " "), err)
	}
	return nil
}
