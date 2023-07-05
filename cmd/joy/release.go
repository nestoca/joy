package main

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/releasing"
	"github.com/nestoca/joy-cli/internal/releasing/list"
	"github.com/nestoca/joy-cli/internal/releasing/promotion"
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
			var filter releasing.Filter
			if releases != "" {
				filter = releasing.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = releasing.NewSpecificReleasesFilter(cfg.Releases.Selected)
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
	var doPreview, doPromote, noPush bool
	var releases string

	cmd := &cobra.Command{
		Use:     "promote",
		Aliases: []string{"prom"},
		Short:   "Promote releases from one environment to another",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Environments.Source == "" || cfg.Environments.Target == "" {
				fmt.Printf("🙏Please run %s to specify source and target promotion environments.", color.InWhite("joy env select"))
				return nil
			}

			// Filtering
			var filter releasing.Filter
			if releases != "" {
				filter = releasing.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = releasing.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			// Options
			opts := promotion.Opts{
				SourceEnv: cfg.Environments.Source,
				TargetEnv: cfg.Environments.Target,
				Push:      !noPush,
				Filter:    filter,
			}

			// Action
			if doPreview {
				opts.Action = promotion.ActionPreview
			} else if doPromote {
				opts.Action = promotion.ActionPromote
			}

			// Filter
			if releases != "" {
				opts.Filter = releasing.NewNamePatternFilter(releases)
			}

			return promotion.Prompt(opts)
		},
	}

	cmd.Flags().BoolVar(&doPreview, "preview", false, "Preview changes (don't prompt for action)")
	cmd.Flags().BoolVar(&doPromote, "promote", false, "Promote changes (don't prompt for action)")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Skip pushing changes (only commit locally)")
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to promote (comma-separated with wildcards, defaults to prompting user)")

	return cmd
}

func NewReleaseSelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose releases to work with",
		Long: `Choose releases to work with.

Only selected releases will be included in releases table and during promotion.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return releasing.Select(cfg.FilePath)
		},
	}
	return cmd
}
