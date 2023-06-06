/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package build

import (
	"github.com/spf13/cobra"
)

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	Use:   "build",
}

func init() {
	// Add sub commands here
	BuildCmd.AddCommand(promoteCmd)

	// Here you will define your flags and configuration settings.
	
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// BuildCmd.PersistentFlags().String("foo", "", "A help for foo")
	
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// BuildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
