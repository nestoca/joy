package diagnose

import (
	"fmt"

	"github.com/nestoca/joy/internal/config"
)

func Diagnose(cliVersion string, cfg *config.Config, builder DiagnosticBuilder) error {
	builder.Start()
	defer builder.End()

	err := diagnoseExecutable(cfg, cliVersion, builder)
	if err != nil {
		return fmt.Errorf("diagnosing executable: %w", err)
	}
	diagnoseDependencies(builder)
	diagnoseConfig(cfg, cliVersion, builder)
	diagnoseCatalog(cfg.CatalogDir, builder)

	return nil
}
