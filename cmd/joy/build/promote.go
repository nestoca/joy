/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package build

import (
	internalBuild "github.com/nestoca/joy-cli/internal/build"
	"github.com/spf13/cobra"
)

// promoteCmd represents the promote command
var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promotes a project to the specified version",
	Long: `Promotes a project to the specified version in the specified environment.

Usage: joy build promote --env <env> <project> <version>`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := cmd.Flag("env").Value.String()
		project := args[0]
		version := args[1]

		return internalBuild.Promote(internalBuild.PromoteArgs{
			Environment: env,
			Project: project,
			Version: version,
		})
	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// promoteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	promoteCmd.Flags().StringP("env", "e", "", "Environment in which to promote this build")
	promoteCmd.MarkFlagRequired("env")
}
