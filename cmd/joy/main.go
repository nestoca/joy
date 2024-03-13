package main

import (
	"os"
	"runtime/debug"
)

// version represents the version of our built application.
// it will be set via ldflags during the build process.
var version string

func main() {
	if version == "" {
		version = debugBuildVersion()
	}

	if err := NewRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}

func debugBuildVersion() string {
	info, _ := debug.ReadBuildInfo()
	return info.Main.Version
}
