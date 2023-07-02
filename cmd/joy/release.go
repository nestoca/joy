package main

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/release/list"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel"},
		Short:   "Manage releases",
	}
	cmd.AddCommand(NewReleaseListCmd())
	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List releases across environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			catalogDir, err := utils.ResolvePath(viper.GetString("catalog-dir"))
			if err != nil {
				return fmt.Errorf("failed to resolve catalog directory path: %w", err)
			}

			return list.List(list.Opts{
				BaseDir: catalogDir,
			})
		},
	}
}
