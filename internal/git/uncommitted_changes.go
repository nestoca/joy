package git

import (
	"fmt"
	"github.com/TwiN/go-color"
	"os/exec"
	"strings"
)

func EnsureNoUncommittedChanges() error {
	cmd := exec.Command("git", "status", "--porcelain")
	outputBytes, err := cmd.Output()
	if err != nil {
		return err
	}

	output := strings.TrimSpace(string(outputBytes))
	if len(output) > 0 {
		return fmt.Errorf("uncommitted changes detected:\n%s", color.InRed(output))
	}

	return nil
}
