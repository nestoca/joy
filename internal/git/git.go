package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nestoca/survey/v2"

	"github.com/nestoca/joy/internal"
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

func Run(dir string, io internal.IO, args []string) error {
	args = append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", args...)

	cmd.Stdin = io.In
	cmd.Stdout = io.Out
	cmd.Stderr = io.Err

	if err := cmd.Run(); err != nil {
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

type UncommittedChangesOptions struct {
	SkipUntrackedFiles bool
}

func GetUncommittedChanges(dir string) ([]string, error) {
	return GetUncommittedChangesWithOpts(dir, UncommittedChangesOptions{})
}

func GetUncommittedChangesWithOpts(dir string, opts UncommittedChangesOptions) ([]string, error) {
	output, err := exec.Command("git", "-C", dir, "status", "--porcelain").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing git status: %s", string(output))
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	var result []string
	for _, value := range strings.Split(trimmed, "\n") {
		if opts.SkipUntrackedFiles && strings.HasPrefix(value, "??") {
			continue
		}
		result = append(result, value)
	}

	return result, nil
}

// IsDirty check if any files in the the git directory specified by dir have been modified.
func IsDirty(dir string, opts UncommittedChangesOptions) (bool, error) {
	changes, err := GetUncommittedChangesWithOpts(dir, opts)
	if err != nil {
		return false, err
	}
	return len(changes) != 0, nil
}

func Stash(dir string) error {
	output, err := exec.Command("git", "-C", dir, "stash").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

func StashApply(dir string) error {
	output, err := exec.Command("git", "-C", dir, "stash", "apply").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

func IsBranchInSyncWithRemote(dir string, branch string) (bool, error) {
	// Fetch the latest changes from the remote
	fetchCmd := exec.Command("git", "-C", dir, "fetch", "origin", branch)
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("fetching from remote: %s", string(fetchOutput))
	}

	// Use git status --porcelain to check the sync status
	statusCmd := exec.Command("git", "-C", dir, "status", "--porcelain", "-b")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking git status: %s", string(statusOutput))
	}

	// Regular expression to match branch status
	branchStatusRegex := regexp.MustCompile(`## [^ ]+( \[(ahead \d+|behind \d+|diverged \d+ and \d+)\])?`)
	matches := branchStatusRegex.FindStringSubmatch(string(statusOutput))

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
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func SwitchBack(dir string) error {
	output, err := exec.Command("git", "-C", dir, "checkout", "-").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

func CreateBranch(dir, name string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", "-b", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating branch %s: %s", name, string(output))
	}
	return nil
}

func Add(dir string, files []string) error {
	args := append([]string{"-C", dir, "add", "--"}, files...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adding files to index: %s", string(output))
	}
	return nil
}

func Commit(dir, message string) error {
	cmd := exec.Command("git", "-C", dir, "commit", "--no-verify", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("committing changes: %s", string(output))
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
		return fmt.Errorf("pushing new branch %s: %s", name, string(output))
	}
	return nil
}

func Pull(dir string, args ...string) error {
	args = append([]string{"-C", dir, "pull"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}
	return nil
}

func Diff(dir string, ref string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "diff", "--name-only", ref)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %q", err, string(output))
	}

	var result []string
	for _, value := range strings.Split(string(output), "\n") {
		if value == "" {
			continue
		}
		result = append(result, value)
	}

	return result, nil
}

func Reset(dir string) error {
	// Check for uncommitted changes
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checking for uncommitted changes: %s", string(output))
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
		return fmt.Errorf("resetting changes: %s", string(output))
	}
	fmt.Println("✅ Uncommitted changes discarded successfully!")
	return nil
}

func GetCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %s", string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCurrentCommit(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting current commit: %s", string(output))
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
