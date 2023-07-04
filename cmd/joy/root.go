package main

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/viper"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "joy",
		Short:        "Manages project, environment and release resources as code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initConfig(cmd.Flag("config").Value.String())
		},
	}
	cmd.PersistentFlags().StringP("config", "c", "~/.joy/config.yaml", "Configuration for the joy-cli containing overrides for the default settings")

	// Core commands
	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"})
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

func initConfig(path string) error {
	configFile, err := utils.ResolvePath(path)
	if err != nil {
		return fmt.Errorf("failed resolve config file path: %w", err)
	}

	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)

	// Overrides can be set using env vars and take precedence over the config file. Useful for CI
	// Will look for env vars prefixed with `JOY_` followed by the config name.
	// Ex: `catalog-dir`'s value would be set to the value of JOY_CATALOG_DIR
	viper.SetEnvPrefix("joy")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("catalog-dir", "~/.joy/catalog")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
}

func changeToCatalogDir() error {
	catalogDir, err := utils.ResolvePath(viper.GetString("catalog-dir"))
	if err != nil {
		return fmt.Errorf("failed to resolve catalog directory path: %w", err)
	}
	err = os.Chdir(catalogDir)
	if err != nil {
		return fmt.Errorf("changing to catalog directory: %w", err)
	}
	return nil
}
