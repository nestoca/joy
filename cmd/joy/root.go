package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nestoca/survey/v2"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewRootCmd(version string) *cobra.Command {
	var (
		configDir        string
		catalogDir       string
		skipVersionCheck bool
		flags            config.GlobalFlags
		setupCmd         = NewSetupCmd(version, &configDir, &catalogDir)
		diagnoseCmd      = NewDiagnoseCmd(version)
		versionCmd       = NewVersionCmd(version)
	)

	cmd := &cobra.Command{
		Use:          "joy",
		Short:        "Manages project, environment and release resources as code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.CalledAs() == "help" {
				return nil
			}

			if cmd != diagnoseCmd && cmd != setupCmd {
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

			if !skipVersionCheck && os.Getenv("JOY_DEV_SKIP_VERSION_CHECK") != "1" {
				if version != "(devel)" && semver.Compare(version, cfg.MinVersion) < 0 {
					err := fmt.Errorf("current version %q is less than required minimum version %q", version, cfg.MinVersion)
					fmt.Println("Please update joy! >> brew update && brew upgrade joy")
					return err
				}

				if !semver.IsValid(version) {
					var ok bool
					prompt := &survey.Confirm{Message: "You are running joy on a development build. Do you wish to continue?"}
					if err := survey.AskOne(prompt, &ok); err != nil {
						return err
					}
					if !ok {
						return errors.New("aborting run")
					}
					fmt.Println()
				}
			}

			cmd.SetContext(config.ToFlagsContext(cmd.Context(), &flags))

			if cmd == setupCmd || cmd == diagnoseCmd || cmd == versionCmd {
				return nil
			}

			cat, err := catalog.Load(cfg.CatalogDir, cfg.KnownChartRefs())
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}
			cmd.SetContext(catalog.ToContext(cmd.Context(), cat))

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "")
	cmd.PersistentFlags().MarkHidden("skip-version-check")

	cmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Directory containing .joyrc config file (defaults to $HOME)")
	cmd.PersistentFlags().StringVar(&catalogDir, "catalog-dir", "", "Directory containing joy catalog of environments, projects and releases (defaults to $HOME/.joy)")

	cmd.PersistentFlags().BoolVar(&flags.SkipCatalogUpdate, "skip-catalog-update", false, "Skip catalog update and dirty check")

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
	cmd.AddCommand(versionCmd)
	cmd.AddCommand(setupCmd)
	cmd.AddCommand(diagnoseCmd)
	cmd.AddCommand(NewExecuteCmd())

	return cmd
}
