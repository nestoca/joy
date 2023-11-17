package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/style"

	"github.com/nestoca/joy/internal/config"
)

func diagnoseExecutable(cfg *config.Config, cliVersion string) (group Group) {
	group.Title = "Executable"
	group.toplevel = true

	group.AddMsg(info, label("Version", cliVersion))

	// Executable version
	if !semver.IsValid(cliVersion) {
		group.AddMsg(warning, fmt.Sprintf("Version is not in semver format and cannot be compared with minimum of %s required by catalog", style.Code(cfg.MinVersion)))
		return
	}

	if semver.Compare(cliVersion, cfg.MinVersion) < 0 {
		group.AddMsg(
			failed,
			fmt.Sprintf("Version does not meet minimum of %s required by catalog", style.Code(cfg.MinVersion)),
			msg(hint, fmt.Sprintf("Update joy using: %s", style.Code("brew upgrade joy"))),
		)
		return
	}

	group.AddMsg(success, fmt.Sprintf("Version meets minimum of %s required by catalog", style.Code(cfg.MinVersion)))

	// Executable path
	execPath, err := os.Executable()
	if err != nil {
		group.AddMsg(failed, "failed to get executable path: "+err.Error())
		return
	}

	absolutePath, err := filepath.Abs(execPath)
	if err != nil {
		group.AddMsg(failed, "failed to get absolute path of executable: "+err.Error())
		return
	}

	group.AddMsg(info, label("File path", absolutePath))
	return
}
