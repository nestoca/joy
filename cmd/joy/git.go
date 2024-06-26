package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
)

func NewGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "git",
		Short:              "Call arbitrary git command against catalog with given arguments",
		Long:               `Call arbitrary git command against catalog with given arguments`,
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			if err := os.Chdir(cfg.CatalogDir); err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}
			return git.Run(cfg.CatalogDir, internal.IoFromCommand(cmd), args)
		},
	}
	return cmd
}

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "pull",
		Short:              "Pull catalog changes from git remote",
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			if err := os.Chdir(cfg.CatalogDir); err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}
			return git.Pull(cfg.CatalogDir, args...)
		},
	}
	return cmd
}

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "push",
		Short:              "Push catalog changes to git remote",
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			if err := os.Chdir(cfg.CatalogDir); err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}
			return git.Push(cfg.CatalogDir, args...)
		},
	}
	return cmd
}

func NewResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset",
		Short:   "Reset all uncommitted catalog changes",
		GroupID: "git",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			if err := os.Chdir(cfg.CatalogDir); err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}
			return git.Reset(cfg.CatalogDir)
		},
	}
	return cmd
}
