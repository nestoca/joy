package main

import (
	"os"
)

// version represents the version of our built application.
// it will be set via ldflags during the build process.
var version string

func main() {
	if err := NewRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}
