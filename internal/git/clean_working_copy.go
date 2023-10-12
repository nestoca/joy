package git

import (
	"bytes"
	"fmt"
	"github.com/nestoca/joy/internal/style"
	"os/exec"
	"strings"
)

func EnsureCleanAndUpToDateWorkingCopy() error {
	cmd := exec.Command("git", "status", "--porcelain")
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checking git status: %s", string(outputBytes))
	}

	output := strings.TrimSpace(string(outputBytes))
	if len(output) > 0 {
		return fmt.Errorf("uncommitted changes detected:\n%s", style.Warning(output))
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
