package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/TwiN/go-color"
	"github.com/nestoca/survey/v2"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/pkg/catalog"
)

type PreRunConfig struct {
	PullCatalog bool
}

type PreRunConfigs map[*cobra.Command]PreRunConfig

func (cfgs PreRunConfigs) PullCatalog(cmd *cobra.Command) {
	cfg := cfgs[cmd]
	cfg.PullCatalog = true
	cfgs[cmd] = cfg
}

func NewRootCmd(version string) *cobra.Command {
	var (
		configDir        string
		catalogDir       string
		skipVersionCheck bool
		skipDevCheck     bool
		flags            config.GlobalFlags
		setupCmd         = NewSetupCmd(version, &configDir, &catalogDir)
		diagnoseCmd      = NewDiagnoseCmd(version)
		versionCmd       = NewVersionCmd(version)
	)

	preRunConfigs := make(PreRunConfigs)

	cmd := &cobra.Command{
		Use:           "joy",
		Short:         "Manages project, environment and release resources as code",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.CalledAs() == "help" {
				return nil
			}

			if !skipDevCheck && (!semver.IsValid(version) || semver.Prerelease(version) != "") {
				var ok bool
				prompt := &survey.Confirm{Message: fmt.Sprintf("You are running joy on a development build: %s\n\nDo you wish to continue?", color.InYellow(version))}
				if err := survey.AskOne(prompt, &ok); err != nil {
					return err
				}
				if !ok {
					return errors.New("aborting run")
				}
				fmt.Println()
			}

			if cmd != diagnoseCmd && cmd != setupCmd {
				if err := dependencies.AllRequiredMustBeInstalled(); err != nil {
					return err
				}
			}

			preRunConfig := preRunConfigs[cmd]

			cfg, err := func() (*config.Config, error) {
				cfg := config.FromContext(cmd.Context())
				if cfg == nil {
					return nil, fmt.Errorf("config not found in context")
				}

				if preRunConfig.PullCatalog {
					if flags.SkipCatalogUpdate {
						fmt.Println("ℹ️ Skipping catalog update.")
					} else {
						if err := git.EnsureCleanAndUpToDateWorkingCopy(cfg.CatalogDir); err != nil {
							return nil, fmt.Errorf("ensuring catalog up to date: %w", err)
						}
						var err error
						cfg, err = config.Load(configDir, cfg.CatalogDir)
						if err != nil {
							return nil, err
						}
					}
				}

				cmd.SetContext(config.ToContext(cmd.Context(), cfg))
				return cfg, nil
			}()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if !skipVersionCheck && os.Getenv("JOY_DEV_SKIP_VERSION_CHECK") != "1" && version != "(devel)" && semver.Compare(version, cfg.MinVersion) < 0 {
				format := "Current version %q is less than required minimum version %q\n\n"
				format += "Please update joy! >> brew update && brew upgrade joy"
				return fmt.Errorf(format, version, cfg.MinVersion)
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
	_ = cmd.PersistentFlags().MarkHidden("skip-version-check")

	cmd.PersistentFlags().BoolVar(&skipDevCheck, "skip-dev-check", false, "")
	_ = cmd.PersistentFlags().MarkHidden("skip-dev-check")

	cmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Directory containing .joyrc config file (defaults to $HOME)")
	cmd.PersistentFlags().StringVar(&catalogDir, "catalog-dir", "", "Directory containing joy catalog of environments, projects and releases (defaults to $HOME/.joy)")

	cmd.PersistentFlags().BoolVar(&flags.SkipCatalogUpdate, "skip-catalog-update", false, "Skip catalog update and dirty check")

	// Core commands
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"})
	cmd.AddCommand(NewEnvironmentCmd(preRunConfigs))
	cmd.AddCommand(NewReleaseCmd(preRunConfigs))
	cmd.AddCommand(NewProjectCmd(preRunConfigs))
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
