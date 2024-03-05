package main

import (
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
)

func NewEnvironmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"environment", "env"},
		Short:   "Manage environments",
		Long:    `Manage environments, such as listing and selecting them.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewEnvironmentSelectCmd())
	return cmd
}

func NewEnvironmentSelectCmd() *cobra.Command {
	var skipCatalogUpdate bool
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose environments to work with",
		Long: `Choose environments to work with and to promote from and to.

Only selected environments will be included in releases table columns.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return environment.ConfigureSelection(cfg.CatalogDir, cfg.FilePath, allFlag, skipCatalogUpdate)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all environments")
	return cmd
}
