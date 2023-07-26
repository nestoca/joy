package main

import (
	"fmt"
	"github.com/nestoca/joy/internal/jac"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/release/list"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/release/selection"
	"github.com/nestoca/joy/internal/style"
	"github.com/spf13/cobra"
)

func NewReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel"},
		Short:   "Manage releases",
		Long:    `Manage releases, such as promoting a release in a given environment.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewReleaseListCmd())
	cmd.AddCommand(NewReleasePromoteCmd())
	cmd.AddCommand(NewReleaseSelectCmd())
	cmd.AddCommand(NewReleasePeopleCmd())
	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	var releases string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List releases across environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Filtering
			var filter release.Filter
			if releases != "" {
				filter = release.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = release.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			return list.List(list.Opts{
				SelectedEnvs: cfg.Environments.Selected,
				Filter:       filter,
			})
		},
	}
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to list (comma-separated with wildcards, defaults to all)")
	return cmd
}

func NewReleasePromoteCmd() *cobra.Command {
	var releases string

	cmd := &cobra.Command{
		Use:     "promote",
		Aliases: []string{"prom"},
		Short:   "Promote releases from one environment to another",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Environments.Source == "" || cfg.Environments.Target == "" {
				fmt.Printf("🙏 Please run %s to specify source and target promotion environments.", style.Code("joy env select"))
				return nil
			}

			// Filtering
			var filter release.Filter
			if releases != "" {
				filter = release.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = release.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			// Options
			opts := promote.Opts{
				SourceEnv: cfg.Environments.Source,
				TargetEnv: cfg.Environments.Target,
				Filter:    filter,
			}

			// Filter
			if releases != "" {
				opts.Filter = release.NewNamePatternFilter(releases)
			}

			return promote.Promote(opts)
		},
	}

	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to promote (comma-separated with wildcards, defaults to prompting user)")

	return cmd
}

func NewReleaseSelectCmd() *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose releases to work with",
		Long: `Choose releases to work with.

Only selected releases will be included in releases table and during promotion.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selection.Select(cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all releases")
	return cmd
}

func NewReleasePeopleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "people",
		Short: "List people owning a release's project via jac cli",
		Long: `List people owning a release's project via jac cli.

Calls 'jac people --group <owner1>,<owner2>...' with the owners of the release's project.

All extra arguments and flags are passed to jac cli as is.

This command requires the jac cli: https://github.com/nestoca/jac
`,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return jac.ListReleasePeople(args)
		},
	}
	return cmd
}
