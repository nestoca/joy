package main

import (
	"github.com/nestoca/joy/internal/git"
	"github.com/spf13/cobra"
)

func NewGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "git",
		Short:   "Call arbitrary git command in catalog dir with given arguments",
		Long:    `Call arbitrary git command in catalog dir with given arguments`,
		GroupID: "git",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return git.Run(args)
		},
	}
	return cmd
}

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pull",
		Short:   "Pull changes from git remote",
		GroupID: "git",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return git.Pull()
		},
	}
	return cmd
}

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push",
		Short:   "Push changes to git remote",
		GroupID: "git",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return git.Push()
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
			return git.Reset()
		},
	}
	return cmd
}
