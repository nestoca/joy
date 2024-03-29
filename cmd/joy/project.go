package main

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/jac"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/project"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "proj"},
		Short:   "Manage projects",
		Long:    `Manage projects, such as listing and selecting them.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewProjectListCmd())
	cmd.AddCommand(NewProjectPeopleCmd())
	cmd.AddCommand(NewProjectOpenCmd())
	cmd.AddCommand(NewProjectLinksCmd())
	return cmd
}

func NewProjectPeopleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owners",
		Short: "List people owning a project via jac cli",
		Long: `List people owning a project via jac cli.

Calls 'jac people --group <owner1>,<owner2>...' with the owners of the project.

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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())
			return jac.ListProjectPeople(cat, args)
		},
	}
	return cmd
}

func NewProjectListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects and their owners",
		Aliases: []string{
			"ls",
		},
		Long: `List projects and their owners.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())
			return project.List(cat)
		},
	}
	return cmd
}

func NewProjectOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open [flags] [project] [link]",
		Aliases: []string{"open", "o"},
		Short:   "Open project link",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			projectName := ""
			if len(args) >= 1 {
				projectName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			projectLinks, err := links.GetProjectLinks(linksProvider, cat, projectName)
			if err != nil {
				return fmt.Errorf("getting project links: %w", err)
			}

			url, err := links.GetOrSelectLinkUrl(projectLinks, linkName)
			if err != nil {
				return err
			}

			return browser.OpenURL(url)
		},
	}

	return cmd
}

func NewProjectLinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "links [flags] [project] [link]",
		Aliases: []string{"links", "link", "lnk"},
		Short:   "List project links",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			projectName := ""
			if len(args) >= 1 {
				projectName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			projectLinks, err := links.GetProjectLinks(linksProvider, cat, projectName)
			if err != nil {
				return fmt.Errorf("getting project links: %w", err)
			}

			return links.PrintLinks(projectLinks, linkName)
		},
	}

	return cmd
}
