package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
)

// shouldSkipCatalogLoad reports commands that only need config, not a parsed catalog tree.
func shouldSkipCatalogLoad(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Name() == "dir" && cmd.Parent() != nil && cmd.Parent().Name() == "catalog"
}

func NewCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "catalog",
		Aliases: []string{"cat"},
		Short:   "Catalog path and metadata",
	}
	cmd.AddCommand(newCatalogDirCmd())
	return cmd
}

func newCatalogDirCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "directory",
		Aliases: []string{"dir"},
		Short:   "Print the absolute catalog directory path",
		Long:    `Print the fully resolved absolute path to the joy catalog directory. The output has no trailing newline, for use in shell scripts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			abs, err := filepath.Abs(cfg.CatalogDir)
			if err != nil {
				return fmt.Errorf("resolving catalog directory: %w", err)
			}
			abs = filepath.Clean(abs)
			_, err = fmt.Fprint(cmd.OutOrStdout(), abs)
			return err
		},
	}
}
