package diagnose

import (
	"fmt"

	"github.com/nestoca/joy/internal/dependencies"

	"github.com/nestoca/joy/internal/config"
)

type Opts struct {
	// CliVersion is the version of the CLI
	CliVersion string

	// Config is the configuration
	Config *config.Config

	// RequiredDependencies are the required dependencies
	RequiredDependencies []*dependencies.Dependency

	// OptionalDependencies are the optional dependencies
	OptionalDependencies []*dependencies.Dependency

	// Builder is the diagnostic builder
	Builder DiagnosticBuilder
}

func Diagnose(opts Opts) error {
	builder := opts.Builder
	builder.Start()
	defer builder.End()

	err := diagnoseExecutable(opts.Config, opts.CliVersion, builder)
	if err != nil {
		return fmt.Errorf("diagnosing executable: %w", err)
	}
	diagnoseDependencies(opts.RequiredDependencies, opts.OptionalDependencies, builder)
	diagnoseConfig(opts.Config, opts.CliVersion, builder)
	diagnoseCatalog(opts.Config.CatalogDir, builder)

	return nil
}
