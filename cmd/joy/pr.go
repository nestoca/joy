package main

import (
	"github.com/spf13/cobra"

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
	var noPrompt, disable bool
	var targetEnv string
	cmd := cobra.Command{
		Use:     "promote",
		Aliases: []string{"prom"},
		Short:   "Auto-promote builds of pull request to given environment",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())

			return promote.
				NewDefaultPromotion(".", cmd.OutOrStdout()).
				Promote(promote.Params{
					Environments: cat.Environments,
					TargetEnv:    targetEnv,
					Disable:      disable,
					NoPrompt:     noPrompt,
				})
		},
	}

	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Do not prompt user for anything")
	cmd.Flags().StringVarP(&targetEnv, "target", "t", "", "Environment to auto-promote builds of pull request to")
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable auto-promotion")
	cmd.MarkFlagsMutuallyExclusive("target", "disable")

	return &cmd
}
