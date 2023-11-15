package main

import (
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/diagnose"
	"github.com/spf13/cobra"
)

func NewDiagnoseCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "diagnose",
		Aliases: []string{"diag"},
		Short:   "Diagnose your joy installation",
		Long:    "Diagnose your joy installation, including the joy binary, configuration, dependencies and catalog git working copy.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			builder := diagnose.NewPrintDiagnosticBuilder()
			err := diagnose.Diagnose(version, cfg, builder)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(builder.String()))
			return err
		},
	}
}
