package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal"

	"github.com/nestoca/joy/internal/config"

	flag "github.com/spf13/pflag"

	"github.com/nestoca/joy/internal/help"
)

// version represents the version of our built application.
// it will be set via ldflags during the build process.
var version string

func main() {
	params := RunParams{
		version: version,
		args:    os.Args[1:],
		io: internal.IO{
			Out: os.Stdout,
			Err: os.Stderr,
			In:  os.Stdin,
		},
		preRunConfigs: make(PreRunConfigs),
	}

	if err := run(params); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type RunParams struct {
	version              string
	args                 []string
	io                   internal.IO
	customizeRootCmdFunc func(rootCmd *cobra.Command) error
	preRunConfigs        PreRunConfigs
}

func run(params RunParams) error {
	version := params.version
	if version == "" {
		version = debugBuildVersion()
	}

	var configDir, catalogDir string
	flags := flag.NewFlagSet("root", flag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.StringVar(&configDir, "config-dir", "", "")
	flags.StringVar(&catalogDir, "catalog-dir", "", "")
	flags.Usage = func() {}
	_ = flags.Parse(params.args)

	cfg, err := config.Load(configDir, catalogDir)
	if err != nil {
		return err
	}

	rootCmd := NewRootCmd(version, params.preRunConfigs)
	rootCmd.SetArgs(params.args)
	rootCmd.SetOut(params.io.Out)
	rootCmd.SetErr(params.io.Err)
	rootCmd.SetIn(params.io.In)
	rootCmd.SetContext(config.ToContext(context.Background(), cfg))
	help.AugmentCommandHelp(rootCmd)

	if params.customizeRootCmdFunc != nil {
		if err := params.customizeRootCmdFunc(rootCmd); err != nil {
			return fmt.Errorf("customizing root command: %w", err)
		}
	}

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
