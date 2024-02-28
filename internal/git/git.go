package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
)

var dependency = &dependencies.Dependency{
	Command:    "git",
	Url:        "https://git-scm.com/downloads",
	IsRequired: true,
}

func init() {
	dependencies.Add(dependency)
}

func Run(dir string, args []string) error {
	args = append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running git command: %w", err)
	}
	return nil
}

func IsValid(dir string) bool {
	// Must have a .git directory directly under given directory
	dotGitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(dotGitDir)
	if err != nil {
		return false
	}

	return exec.Command("git", "-C", dir, "status").Run() == nil
}

func GetUncommittedChanges(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing git status: %s", string(outputBytes))
	}
	trimmed := strings.TrimSpace(string(outputBytes))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func GetDefaultBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "symbolic-ref", "refs/remotes/origin/HEAD")
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	longName := strings.TrimSpace(string(outputBytes))
	return strings.TrimPrefix(longName, "refs/remotes/origin/"), nil
}

func IsBranchInSyncWithRemote(dir string, branch string) (bool, error) {
	// Fetch the latest changes from the remote
	fetchCmd := exec.Command("git", "-C", dir, "fetch", "origin", branch)
	if err := fetchCmd.Run(); err != nil {
		fmt.Println("Error fetching from remote:", err)
		return false, err
	}

	// Use git status --porcelain to check the sync status
	statusCmd := exec.Command("git", "-C", dir, "status", "--porcelain", "-b")
	var out bytes.Buffer
	statusCmd.Stdout = &out
	if err := statusCmd.Run(); err != nil {
		fmt.Println("Error checking git status:", err)
		return false, err
	}

	// Regular expression to match branch status
	branchStatusRegex := regexp.MustCompile(`## [^ ]+( \[(ahead \d+|behind \d+|diverged \d+ and \d+)\])?`)
	matches := branchStatusRegex.FindStringSubmatch(out.String())

	// Check if the branch is either ahead, behind or diverged
	if len(matches) > 1 && matches[1] != "" {
		return false, nil
	}
	return true, nil
}

func Checkout(dir, branch string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("checkint out branch %s: %w", branch, err)
	}
	return nil
}

func CreateBranch(dir, name string) error {
	// Create and checkout branch
	cmd := exec.Command("git", "-C", dir, "checkout", "-b", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("creating branch %s: %w", name, err)
	}
	return nil
}

func Add(dir string, files []string) error {
	args := append([]string{"-C", dir, "add", "--"}, files...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("adding files to index: %w", err)
	}
	return nil
}

func Commit(dir, message string) error {
	cmd := exec.Command("git", "-C", dir, "commit", "--no-verify", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("committing changes: %w", err)
	}
	return nil
}

func Push(dir string, args ...string) error {
	args = append([]string{"-C", dir, "push"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pushing changes: %w", err)
	}
	return nil
}

func PushNewBranch(dir, name string) error {
	// Set upstream to origin
	cmd := exec.Command("git", "-C", dir, "push", "-u", "origin", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("pushing new branch %s: %w", name, err)
	}
	return nil
}

func Pull(dir string, args ...string) error {
	args = append([]string{"-C", dir, "pull"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}
	return nil
}

func Reset(dir string) error {
	// Check for uncommitted changes
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("checking for uncommitted changes: %w", err)
	}
	outputText := strings.TrimSpace(string(output))
	if len(outputText) == 0 {
		fmt.Println("🤷 No uncommitted changes were found")
		return nil
	}

	// Ask for confirmation
	fmt.Printf("🔥 Uncommitted changes detected:\n%s", style.Warning(string(output)))
	confirm := false
	prompt := &survey.Confirm{
		Message: "Are you sure you want discard all uncommitted changes?",
		Default: false,
	}
	err = survey.AskOne(prompt, &confirm)
	if err != nil {
		return fmt.Errorf("asking for confirmation: %w", err)
	}
	if !confirm {
		fmt.Println("❌ Reset cancelled by user")
		return nil
	}

	// Perform reset
	cmd = exec.Command("git", "-C", dir, "reset", "--hard")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("resetting changes: %w", err)
	}
	fmt.Println("✅ Uncommitted changes discarded successfully!")
	return nil
}

func GetCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCurrentCommit(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func Fetch(dir string) error {
	cmd := exec.Command("git", "fetch")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error: %s", string(output))
	}

	return nil
}

func FetchTags(dir string) error {
	cmd := exec.Command("git", "fetch", "--tags")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error: %s", string(output))
	}

	return nil
}
