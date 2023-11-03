package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
)

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	var (
		configDir  string
		catalogDir string
		setupCmd   = NewSetupCmd(&configDir, &catalogDir)
	)

	cmd := &cobra.Command{
		Use:          "joy",
		Short:        "Manages project, environment and release resources as code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd != setupCmd {
				dependencies.AllRequiredMustBeInstalled()
			}

			cfg, err := config.Load(configDir, catalogDir)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			cmd.SetContext(config.ToContext(cmd.Context(), cfg))

			if cmd == setupCmd {
				return nil
			}

			return config.CheckCatalogDir(cfg.CatalogDir)
		},
	}

	cmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Directory containing .joyrc config file (defaults to $HOME)")
	cmd.PersistentFlags().StringVar(&catalogDir, "catalog-dir", "", "Directory containing joy catalog of environments, projects and releases (defaults to $HOME/.joy)")

	// Core commands
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"})
	cmd.AddCommand(NewEnvironmentCmd())
	cmd.AddCommand(NewReleaseCmd())
	cmd.AddCommand(NewProjectCmd())
	cmd.AddCommand(NewPRCmd())
	cmd.AddCommand(NewBuildCmd())

	// Catalog git commands
	cmd.AddGroup(&cobra.Group{ID: "git", Title: "Catalog git commands"})
	cmd.AddCommand(NewGitCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewResetCmd())

	// Additional commands
	cmd.AddCommand(NewSecretCmd())
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(setupCmd)

	return cmd
}
