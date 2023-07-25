package main

import (
	"github.com/nestoca/joy/internal/environment/selection"
	"github.com/spf13/cobra"
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
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose environments to work with",
		Long: `Choose environments to work with and to promote from and to.

Only selected environments will be included in releases table columns.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selection.Select(cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all environments")
	return cmd
}
