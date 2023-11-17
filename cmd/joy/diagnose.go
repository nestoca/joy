package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/diagnostics"
)

func NewDiagnoseCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "diagnose",
		Aliases: []string{"diag"},
		Short:   "Diagnose your joy installation",
		Long:    "Diagnose your joy installation, including the joy binary, configuration, dependencies and catalog git working copy.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			result := diagnostics.Evaluate(version, cfg)
			stats := result.Stats()

			fmt.Fprint(cmd.OutOrStdout(), result)

			if stats.Failed+stats.Warnings > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\n🚨 Diagnostics completed with %d error(s) and %d warning(s)\n", stats.Failed, stats.Warnings)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\n🚀 All systems nominal. Houston, we're cleared for launch!")
			return nil
		},
	}
}
