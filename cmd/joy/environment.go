package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewEnvironmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"environment", "env"},
		Short:   "Manage environments",
		Long:    `Manage environments, such as listing and selecting them.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewEnvironmentSelectCmd())
	cmd.AddCommand(NewEnvironmentLinksCmd())
	cmd.AddCommand(NewEnvironmentOpenCmd())
	return cmd
}

func NewEnvironmentSelectCmd() *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose environments to work with",
		Long: `Choose environments to work with and to promote from and to.

Only selected environments will be included in releases table columns.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return environment.ConfigureSelection(cfg.CatalogDir, cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all environments")
	return cmd
}

func NewEnvironmentOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open [flags] [environment] [link]",
		Aliases: []string{"open", "o"},
		Short:   "Open environment link",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			envName := ""
			if len(args) >= 1 {
				envName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			cat, err := catalog.Load(catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
			})
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			envLinks, err := links.GetEnvironmentLinks(linksProvider, cat, envName)
			if err != nil {
				return fmt.Errorf("getting project links: %w", err)
			}

			url, err := links.GetOrSelectLinkUrl(envLinks, linkName)
			if err != nil {
				return err
			}

			return links.OpenUrl(url)
		},
	}

	return cmd
}

func NewEnvironmentLinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "links [flags] [environment] [link]",
		Aliases: []string{"links", "link", "lnk"},
		Short:   "List environment links",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			envName := ""
			if len(args) >= 1 {
				envName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			cat, err := catalog.Load(catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
			})
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			envLinks, err := links.GetEnvironmentLinks(linksProvider, cat, envName)
			if err != nil {
				return fmt.Errorf("getting project links: %w", err)
			}

			return links.PrintLinks(envLinks, linkName)
		},
	}

	return cmd
}
