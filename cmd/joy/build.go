package main

import (
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/build"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
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
	var (
		chartVersion  string
		excludeLabels []string
	)

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote a project to given version",
		Long: `Promote a project to given version in given environment.
Typically called at the end of a CI pipeline to promote a new build to default target environment.

Usage: joy build promote [--chart-version <chart-version>] [--exclude-label <key[=value]>]... <env> <project> <version>`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := args[0]
			project := args[1]
			version := args[2]

			cat := catalog.FromContext(cmd.Context())
			cat.WithEnvironments([]string{env})

			return build.Promote(build.Opts{
				Catalog:       cat,
				Environment:   env,
				Project:       project,
				Version:       version,
				Writer:        yml.DiskWriter,
				ChartVersion:  chartVersion,
				ExcludeLabels: excludeLabels,
			})
		},
	}

	cmd.Flags().StringVar(&chartVersion, "chart-version", "", "(optional) Chart version to promote")
	cmd.Flags().StringArrayVar(&excludeLabels, "exclude-label", nil, "(optional, repeatable) Skip releases carrying this metadata label. Format: `key` or `key=value`; a bare key matches any value (e.g. --exclude-label nesto.ca/preview)")

	return cmd
}
