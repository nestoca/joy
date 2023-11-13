package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
)

func NewRootCmd(version string) *cobra.Command {
	var (
		configDir        string
		catalogDir       string
		skipVersionCheck bool
		setupCmd         = NewSetupCmd(&configDir, &catalogDir)
	)

	cmd := &cobra.Command{
		Use:          "joy",
		Short:        "Manages project, environment and release resources as code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd != setupCmd {
				if err := dependencies.AllRequiredMustBeInstalled(); err != nil {
					return err
				}
			}

			cfg, err := func() (*config.Config, error) {
				if cfg := config.FromContext(cmd.Context()); cfg != nil {
					return cfg, nil
				}

				cfg, err := config.Load(configDir, catalogDir)
				if err != nil {
					return nil, err
				}

				cmd.SetContext(config.ToContext(cmd.Context(), cfg))
				return cfg, err
			}()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if !skipVersionCheck {
				if version != "" && semver.Compare(version, cfg.MinVersion) < 0 {
					return fmt.Errorf("current version %q is less than required minimum version %q. Please update joy", version, cfg.MinVersion)
				}
				if !semver.IsValid(version) {
					var ok bool
					prompt := &survey.Confirm{Message: "you are running joy on a development build (not recommended). Do you wish to continue?"}
					if err := survey.AskOne(prompt, &ok); err != nil || !ok {
						return err
					}
				}
			}

			if cmd == setupCmd {
				return nil
			}

			return config.CheckCatalogDir(cfg.CatalogDir)
		},
	}

	cmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "")
	cmd.PersistentFlags().MarkHidden("skip-version-check")

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
	cmd.AddCommand(NewVersionCmd(version))
	cmd.AddCommand(setupCmd)

	return cmd
}
