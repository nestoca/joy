package diagnose

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/style"

	"github.com/nestoca/joy/internal/config"
)

func diagnoseExecutable(cfg *config.Config, cliVersion string, builder DiagnosticBuilder) error {
	builder.StartDiagnostic("Executable")
	defer builder.EndDiagnostic()

	// Executable version
	AddLabelAndInfo(builder, "Version", "%s", cliVersion)
	if semver.IsValid(cliVersion) {
		if semver.Compare(cliVersion, cfg.MinVersion) < 0 {
			builder.AddError("Version does not meet minimum of %s required by catalog", style.Code(cfg.MinVersion))
			builder.AddRecommendation("Update joy using: %s", style.Code("brew upgrade joy"))
		} else {
			builder.AddSuccess("Version meets minimum of %s required by catalog", style.Code(cfg.MinVersion))
		}
	} else {
		builder.AddWarning("Version is not in semver format and cannot be compared with minimum of %s required by catalog", style.Code(cfg.MinVersion))
	}

	// Executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}
	absolutePath, err := filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("getting absolute path of %s: %w", execPath, err)
	}
	AddLabelAndInfo(builder, "File path", "%s", absolutePath)

	return nil
}
