package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/jac"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/release/list"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/pkg/catalog"
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
	cmd.AddCommand(NewReleaseRenderCmd())
	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	var releases string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List releases across environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			// Filtering
			var filter filtering.Filter
			if releases != "" {
				filter = filtering.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			return list.List(list.Opts{
				CatalogDir:   cfg.CatalogDir,
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
	var sourceEnv, targetEnv string
	var autoMerge bool

	cmd := &cobra.Command{
		Use:     "promote [flags] [releases]",
		Aliases: []string{"prom"},
		Short:   "Promote releases across environments",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			// Filtering
			var filter filtering.Filter
			if releases != "" {
				filter = filtering.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				LoadEnvs:        true,
				LoadReleases:    true,
				SortEnvsByOrder: true,
				ReleaseFilter:   filter,
			}
			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			// Resolve source and target environments
			sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, sourceEnv)
			if err != nil {
				return err
			}
			targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, targetEnv)
			if err != nil {
				return err
			}

			// Resolve environments selected by user via `joy env select`
			selectedEnvironments := v1alpha1.GetEnvironmentsByNames(cat.Environments, cfg.Environments.Selected)

			// Perform promotion
			opts := promote.Opts{
				Catalog:              cat,
				SourceEnv:            sourceEnv,
				TargetEnv:            targetEnv,
				ReleasesFiltered:     filter != nil,
				AutoMerge:            autoMerge,
				SelectedEnvironments: selectedEnvironments,
			}

			_, err = promote.NewDefaultPromotion(cfg.CatalogDir).Promote(opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&sourceEnv, "source", "s", "", "Source environment (interactive if not specified)")
	cmd.Flags().StringVarP(&targetEnv, "target", "t", "", "Target environment (interactive if not specified)")
	cmd.Flags().BoolVar(&autoMerge, "auto-merge", false, "Add auto-merge label to release PR")
	addArgumentsToUsage(cmd, "releases", "Comma-separated list of releases (interactive if not specified)")

	return cmd
}

// addArgumentsToUsage adds positional arguments and their descriptions to the usage template of a command.
func addArgumentsToUsage(cmd *cobra.Command, argumentsAndDescriptions ...string) {
	var builder strings.Builder
	builder.WriteString("Arguments:\n")
	for i := 0; i < len(argumentsAndDescriptions)-1; i += 2 {
		builder.WriteString(fmt.Sprintf("  %-21s %s\n", argumentsAndDescriptions[i], argumentsAndDescriptions[i+1]))
	}
	globalSectionPattern := regexp.MustCompile(`(?m)^Global Flags:`)
	cmd.SetUsageTemplate(globalSectionPattern.ReplaceAllString(cmd.UsageTemplate(), builder.String()+"\n$0"))
}

func NewReleaseSelectCmd() *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Select releases to include in listings and promotions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return release.ConfigureSelection(cfg.CatalogDir, cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all releases (non-interactive)")
	return cmd
}

func NewReleasePeopleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owners",
		Short: "List people owning a release's project via jac cli",
		Long: `List people owning a release's project via jac cli.

Calls 'jac people --group <owner1>,<owner2>...' with the owners of the release's project.

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
			return jac.ListReleasePeople(cfg.CatalogDir, args)
		},
	}
	return cmd
}

func NewReleaseRenderCmd() *cobra.Command {
	var (
		env   string
		color bool
	)

	cmd := &cobra.Command{
		Use:   "render [release]",
		Short: "render kubernetes manifests from joy release",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				LoadReleases:    true,
				LoadEnvs:        true,
				SortEnvsByOrder: true,
				ResolveRefs:     true,
			}

			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			var releaseName string
			if len(args) == 1 {
				releaseName = args[0]
			}

			io := internal.IO{
				Out: cmd.OutOrStdout(),
				Err: cmd.ErrOrStderr(),
				In:  cmd.InOrStdin(),
			}

			return render.Render(cmd.Context(), render.RenderOpts{
				Env:           env,
				Release:       releaseName,
				DefaultChart:  cfg.DefaultChart,
				CacheDir:      cfg.JoyCache,
				ChartMappings: cfg.ChartMappings,
				Catalog:       cat,
				IO:            io,
				Helm:          helm.CLI{IO: io},
				Color:         color,
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "environment to select release from.")
	cmd.Flags().BoolVar(&color, "color", term.IsTerminal(int(os.Stdout.Fd())), "toggle output with color")
	return cmd
}
