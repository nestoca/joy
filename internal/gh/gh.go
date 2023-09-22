package gh

import (
	"errors"
	"fmt"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// CreatePullRequest creates a pull request.
func CreatePullRequest(args ...string) error {
	err := EnsureInstalledAndAuthorized()
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

func EnsureInstalledAndAuthorized() error {
	cmd := exec.Command("which", "gh")
	err := cmd.Run()
	if err != nil {
		fmt.Println("🤓 This command requires the gh cli.\nSee: https://github.com/cli/cli")
		return errors.New("missing gh cli dependency")
	}

	cmd = exec.Command("gh", "auth", "status")
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Check if authorized
	tokenRegex := regexp.MustCompile(`Token: .+`)
	if !tokenRegex.MatchString(outputStr) {
		fmt.Printf("🔐 Please run %s to authorize the gh cli.\n", style.Code("gh auth login"))
		return errors.New("gh cli not authorized")
	}

	// Check if user has required scopes
	scopesRegex := regexp.MustCompile(`Token scopes: .*?\b(read:org)\b.*?\b(repo)\b.*`)
	if !scopesRegex.MatchString(outputStr) {
		fmt.Printf("🔐 Please ensure you have the following permission scopes: %s, %s\n", style.Code("read:org"), style.Code("repo"))
		return errors.New("gh cli token missing required scopes")
	}

	return nil
}
