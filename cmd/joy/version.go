package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display joy cli version",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}
