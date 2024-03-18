package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/diagnostics"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewDiagnoseCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "diagnose",
		Aliases: []string{"diag"},
		Short:   "Diagnose your joy installation",
		Long:    "Diagnose your joy installation, including the joy binary, configuration, dependencies and catalog git working copy.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			_, err := fmt.Fprintln(cmd.OutOrStdout(), diagnostics.OutputWithGlobalStats(diagnostics.Evaluate(version, cfg, cat)))
			return err
		},
	}
}
