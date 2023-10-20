package main

import (
	"fmt"
	"github.com/nestoca/joy/internal/git"
	"github.com/spf13/cobra"
	"os"
)

// changeToCatalogDir changes the current directory to the catalog, for commands
// that need to be run from there.
func changeToCatalogDir() error {
	err := os.Chdir(cfg.CatalogDir)
	if err != nil {
		return fmt.Errorf("changing to catalog directory: %w", err)
	}
	return nil
}

func NewGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "git",
		Short:              "Call arbitrary git command in catalog dir with given arguments",
		Long:               `Call arbitrary git command in catalog dir with given arguments`,
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeToCatalogDir(); err != nil {
				return err
			}
			return git.Run(cfg.CatalogDir, args)
		},
	}
	return cmd
}

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "pull",
		Short:              "Pull changes from git remote",
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeToCatalogDir(); err != nil {
				return err
			}
			return git.Pull(cfg.CatalogDir, args...)
		},
	}
	return cmd
}

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "push",
		Short:              "Push changes to git remote",
		GroupID:            "git",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeToCatalogDir(); err != nil {
				return err
			}
			return git.Push(cfg.CatalogDir, args...)
		},
	}
	return cmd
}

func NewResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset",
		Short:   "Reset all uncommitted git changes",
		GroupID: "git",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeToCatalogDir(); err != nil {
				return err
			}
			return git.Reset(cfg.CatalogDir)
		},
	}
	return cmd
}
