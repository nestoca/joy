package git

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/TwiN/go-color"
	"os"
	"os/exec"
	"strings"
)

func Run(args []string) error {
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

func Checkout(name string) error {
	cmd := exec.Command("git", "checkout", name)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("checkint out branch %s: %w", name, err)
	}
	return nil
}

func CreateBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("creating branch %s: %w", name, err)
	}
	return nil
}

func Add(files []string) error {
	args := append([]string{"add", "--"}, files...)
	cmd := exec.Command("git", args...)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("adding files to index: %w", err)
	}
	return nil
}

func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}
	return nil
}

func Push() error {
	cmd := exec.Command("git", "push")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pushing changes: %w", err)
	}
	return nil
}

func Pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}
	return nil
}

func Reset() error {
	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	outputBytes, err := cmd.Output()
	if err != nil {
		return err
	}
	output := strings.TrimSpace(string(outputBytes))
	if len(output) == 0 {
		fmt.Println("🤷No uncommitted changes were found")
		return nil
	}

	// Ask for confirmation
	fmt.Printf("🔥Uncommitted changes detected:\n%s", color.InRed(output))
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
		fmt.Println("❌Reset cancelled by user")
		return nil
	}

	// Perform reset
	cmd = exec.Command("git", "reset", "--hard")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("resetting changes: %w", err)
	}
	fmt.Println("✅Uncommitted changes discarded successfully!")
	return nil
}
