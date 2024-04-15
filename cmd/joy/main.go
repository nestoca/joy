package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/nestoca/joy/internal/config"

	flag "github.com/spf13/pflag"

	"github.com/nestoca/joy/internal/help"
)

// version represents the version of our built application.
// it will be set via ldflags during the build process.
var version string

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	if version == "" {
		version = debugBuildVersion()
	}

	var configDir, catalogDir string
	flags := flag.NewFlagSet("root", flag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.StringVar(&configDir, "config-dir", "", "")
	flags.StringVar(&catalogDir, "catalog-dir", "", "")
	_ = flags.Parse(os.Args[1:])

	rootCmd := NewRootCmd(version)
	cfg, err := config.Load(configDir, catalogDir)
	if err != nil {
		return err
	}
	rootCmd.SetContext(config.ToContext(context.Background(), cfg))
	help.AugmentCommandHelp(rootCmd)

	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		err = help.WrapError(cmd, err)
		return err
	}
	return nil
}

func debugBuildVersion() string {
	info, _ := debug.ReadBuildInfo()
	return info.Main.Version
}
