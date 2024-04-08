package main

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewEnvironmentCmd(preRunConfigs PreRunConfigs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"environment", "env"},
		Short:   "Manage environments",
		Long:    `Manage environments, such as listing and selecting them.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewEnvironmentSelectCmd(preRunConfigs))
	cmd.AddCommand(NewEnvironmentLinksCmd())
	cmd.AddCommand(NewEnvironmentOpenCmd())
	return cmd
}

func NewEnvironmentSelectCmd(preRunConfigs PreRunConfigs) *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Choose environments to work with",
		Long: `Choose environments to work with and to promote from and to.

Only selected environments will be included in releases table columns.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())
			return environment.ConfigureSelection(cat, cfg.FilePath, allFlag)
		},
	}

	preRunConfigs.PullCatalog(cmd)

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
			cat := catalog.FromContext(cmd.Context())
			envName := ""
			if len(args) >= 1 {
				envName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
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

			return browser.OpenURL(url)
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
			cat := catalog.FromContext(cmd.Context())

			envName := ""
			if len(args) >= 1 {
				envName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			envLinks, err := links.GetEnvironmentLinks(linksProvider, cat, envName)
			if err != nil {
				return fmt.Errorf("getting project links: %w", err)
			}

			output, err := links.FormatLinks(envLinks, linkName)
			if err != nil {
				return fmt.Errorf("formatting links: %w", err)
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}

	return cmd
}
