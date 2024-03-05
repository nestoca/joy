package main

import (
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/jac"
	"github.com/nestoca/joy/internal/project"
)

func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "proj"},
		Short:   "Manage projects",
		Long:    `Manage projects, such as listing and selecting them.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewProjectListCmd())
	cmd.AddCommand(NewProjectPeopleCmd())
	return cmd
}

func NewProjectPeopleCmd() *cobra.Command {
	var skipCatalogUpdate bool
	cmd := &cobra.Command{
		Use:   "owners",
		Short: "List people owning a project via jac cli",
		Long: `List people owning a project via jac cli.

Calls 'jac people --group <owner1>,<owner2>...' with the owners of the project.

All extra arguments and flags are passed to jac cli as is.

This command requires the jac cli: https://github.com/nestoca/jac
`,
		Aliases: []string{
			"owner",
			"own",
			"people",
		},
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return jac.ListProjectPeople(cfg.CatalogDir, args, skipCatalogUpdate)
		},
	}
	return cmd
}

func NewProjectListCmd() *cobra.Command {
	var skipCatalogUpdate bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects and their owners",
		Aliases: []string{
			"ls",
		},
		Long: `List projects and their owners.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return project.List(cfg.CatalogDir, skipCatalogUpdate)
		},
	}
	return cmd
}
