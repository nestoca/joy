package diagnose

import (
	"fmt"
	"os"

	"github.com/nestoca/joy/internal/config"
)

func diagnoseConfig(cfg *config.Config, cliVersion string, builder DiagnosticBuilder) {
	builder.StartDiagnostic("Config")
	defer builder.EndDiagnostic()

	if _, err := os.Stat(cfg.FilePath); os.IsNotExist(err) {
		AddLabelAndError(builder, "File does not exist", cfg.FilePath)
		return
	} else {
		AddLabelAndSuccess(builder, "File exists", cfg.FilePath)
	}

	formatSelection := func(selected []string) string {
		if len(selected) == 0 {
			return "<all>"
		}
		return fmt.Sprintf("%d", len(selected))
	}

	AddLabelAndInfo(builder, "Selected environments", formatSelection(cfg.Environments.Selected))
	AddLabelAndInfo(builder, "Selected releases", formatSelection(cfg.Releases.Selected))
}
