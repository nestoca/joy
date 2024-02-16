package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pr",
		Short:   "Manage pull requests",
		Long:    `Manage pull requests, such as auto-promoting builds of a pull request to given environment.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewPRPromoteCmd())
	return cmd
}

func NewPRPromoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "promote",
		Aliases: []string{"prom"},
		Short:   "Auto-promote builds of pull request to given environment",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
			}
			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			return promote.
				NewDefaultPromotion(".").
				Promote(cat.Environments)
		},
	}
}
