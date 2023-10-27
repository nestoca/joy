package git

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
	"strings"
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
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
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
