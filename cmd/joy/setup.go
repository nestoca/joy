package main

import (
	"github.com/nestoca/joy/internal/setup"
	"github.com/spf13/cobra"
)

func NewSetupCmd() *cobra.Command {
	var catalogRepo string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup joy for first time use",
		Long: `Setup joy for first time use.

It prompts user for catalog directory, optionally cloning it if needed, creates config file and checks for required and optional dependencies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.Setup(configDir, catalogDir, catalogRepo)
		},
	}
	cmd.Flags().StringVar(&catalogRepo, "catalog-repo", "", "URL of catalog git repo (defaults to prompting user)")
	return cmd
}