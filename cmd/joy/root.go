package main

import (
	"fmt"
	"github.com/nestoca/joy/internal/config"
	"github.com/spf13/cobra"
	"os"
)

var cfg *config.Config

func main() {

	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	var configDir, catalogDir string
	cmd := &cobra.Command{
		Use:          "joy",
		Short:        "Manages project, environment and release resources as code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.Load(configDir, catalogDir)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Make catalog dir the current directory, as all commands
			// need to be run from there for loading catalog and executing
			// git commands.
			err = os.Chdir(cfg.CatalogDir)
			if err != nil {
				return fmt.Errorf("changing to catalog directory: %w", err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Directory containing .joyrc config file (defaults to $HOME)")
	cmd.PersistentFlags().StringVar(&catalogDir, "catalog-dir", "", "Directory containing joy catalog of environments, projects and releases (defaults to $HOME/.joy)")

	// Core commands
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"})
	cmd.AddCommand(NewEnvironmentCmd())
	cmd.AddCommand(NewReleaseCmd())
	cmd.AddCommand(NewBuildCmd())

	// Git-oriented commands
	cmd.AddGroup(&cobra.Group{ID: "git", Title: "Git-oriented commands"})
	cmd.AddCommand(NewGitCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewResetCmd())

	return cmd
}
