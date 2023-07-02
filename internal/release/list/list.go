package list

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/release/cross"
	"path/filepath"
)

type Opts struct {
	// BaseDir is the base directory to load releases from.
	BaseDir string
}

func List(opts Opts) error {
	environmentsDir := filepath.Join(opts.BaseDir, "environments")
	environments, err := environment.LoadAll(environmentsDir)
	if err != nil {
		return fmt.Errorf("loading environments: %w", err)
	}

	list, err := cross.Load(environmentsDir, environments)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	list.Print()
	return nil
}
