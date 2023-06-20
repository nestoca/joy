package main

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/viper"
	"os"
	"strings"

	"github.com/nestoca/joy-cli/cmd/joy/build"
	"github.com/spf13/cobra"
)

func main() {
	err := createCLI().Execute()
	if err != nil {
		os.Exit(1)
	}
}

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "joy",
	Short: "CLI for managing Joy resources",
	// TODO: Long description
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return initConfig(cmd.Flag("config").Value.String())
	},
}

func createCLI() *cobra.Command {
	// Add subcommands here
	rootCmd.AddCommand(build.Cmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringP("config", "c", "~/.joy/config.yaml", "Configuration for the joy-cli containing overrides for the default settings")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	return rootCmd
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
