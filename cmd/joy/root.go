package main

import (
	"fmt"
	"os"

	"github.com/nestoca/joy-cli/cmd/joy/build"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "joy",
	Short: "CLI for managing Joy resources",
	// TODO: Long description
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := utils.ResolvePath(cmd.Flag("config").Value.String())
		if err != nil {
			return fmt.Errorf("failed resolve config file path: %w", err)
		}

		viper.SetConfigType("yaml")
		viper.SetConfigFile(configFile)

		viper.SetDefault("catalogDir", "~/.joy/catalog")

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				return fmt.Errorf("failed to read config: %w", err)
			}
		}

		return nil
	},
}

func createCLI() *cobra.Command {
	// Add subcommands here
	rootCmd.AddCommand(build.Cmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringP("config", "c", "~/.joy/config.yaml", "config file (default is ~/.joy/config.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	return rootCmd
}

func main() {
	err := createCLI().Execute()
	if err != nil {
		os.Exit(1)
	}
}
