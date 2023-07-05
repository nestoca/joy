package git

import (
	"bytes"
	"fmt"
	"github.com/TwiN/go-color"
	"os/exec"
	"strings"
)

func EnsureCleanAndUpToDateWorkingCopy() error {
	cmd := exec.Command("git", "status", "--porcelain")
	outputBytes, err := cmd.Output()
	if err != nil {
		return err
	}

	output := strings.TrimSpace(string(outputBytes))
	if len(output) > 0 {
		return fmt.Errorf("uncommitted changes detected:\n%s", color.InRed(output))
	}

	buf := bytes.Buffer{}
	cmd = exec.Command("git", "pull")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes:\n%s", buf.String())
	}
	return nil
}
