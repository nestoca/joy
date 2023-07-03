package main

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/release"
	"github.com/nestoca/joy-cli/internal/release/list"
	"github.com/nestoca/joy-cli/internal/release/promote"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

func NewReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel"},
		Short:   "Manage releases",
	}
	cmd.AddCommand(NewReleaseListCmd())
	cmd.AddCommand(NewReleasePromoteCmd())
	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	var releases string
	cmd := &cobra.Command{
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
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to list (comma-separated with wildcards, defaults to all)")
	return cmd
}

func NewReleasePromoteCmd() *cobra.Command {
	var preview, apply, applyNoCommit bool
	var releases string
	cmd := &cobra.Command{
		Use:     "promote",
		Aliases: []string{"prom"},
		Short:   "Promote releases from one environment to another",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			catalogDir, err := utils.ResolvePath(viper.GetString("catalog-dir"))
			if err != nil {
				return fmt.Errorf("failed to resolve catalog directory path: %w", err)
			}
			err = os.Chdir(catalogDir)
			if err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}

			opts := promote.Opts{
				BaseDir:   "",
				SourceEnv: args[0],
				TargetEnv: args[1],
			}

			if preview {
				opts.Action = promote.ActionPreview
			} else if apply {
				opts.Action = promote.ActionApply
			} else if applyNoCommit {
				opts.Action = promote.ActionApplyNoCommit
			}

			if releases != "" {
				opts.Filter = release.NewNamePatternFilter(releases)
			}

			return promote.Promote(opts)
		},
	}
	cmd.Flags().BoolVar(&preview, "preview", false, "Preview changes")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply changes")
	cmd.Flags().BoolVar(&applyNoCommit, "apply-no-commit", false, "Apply changes but don't commit")
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to promote (comma-separated with wildcards, defaults to prompting user)")
	return cmd
}
