package git

import (
	"fmt"
	"os"
	"os/exec"
)

func Run(args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running git command: %w", err)
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
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes: %w", err)
	}
	return nil
}

func Reset() error {
	cmd := exec.Command("git", "reset", "--hard")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("resetting changes: %w", err)
	}
	return nil
}
