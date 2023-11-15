package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nestoca/joy/internal/style"
)

func EnsureCleanAndUpToDateWorkingCopy(dir string) error {
	changes, err := GetUncommittedChanges(dir)
	if err != nil {
		return fmt.Errorf("getting uncommitted changes: %w", err)
	}
	if len(changes) > 0 {
		return fmt.Errorf("uncommitted changes detected:\n%s", style.Warning(strings.Join(changes, "\n")))
	}

	buf := bytes.Buffer{}
	cmd := exec.Command("git", "-C", dir, "pull")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("pulling changes:\n%s", buf.String())
	}
	return nil
}
