package main

import (
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/build"
	"github.com/nestoca/joy/internal/config"
)

func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Manage builds",
		Long:    `Manage builds, such as promoting a build in a given environment.`,
		GroupID: "core",
		Args:    cobra.ExactArgs(2),
	}
	cmd.AddCommand(NewBuildPromoteCmd())
	return cmd
}

func NewBuildPromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote a project to given version",
		Long: `Promote a project to given version in given environment.
Typically called at the end of a CI pipeline to promote a new build to default target environment.

Usage: joy build promote <env> <project> <version>`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := args[0]
			project := args[1]
			version := args[2]

			cfg := config.FromContext(cmd.Context())

			return build.Promote(build.Opts{
				CatalogDir:  cfg.CatalogDir,
				Environment: env,
				Project:     project,
				Version:     version,
			})
		},
	}

	return cmd
}
