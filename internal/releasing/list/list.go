package list

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/environment"
	"github.com/nestoca/joy-cli/internal/releasing"
	"path/filepath"
)

type Opts struct {
	// BaseDir is the base directory to load releases from.
	BaseDir string

	// Filter specifies releases to list.
	// Optional, defaults to listing all releases.
	Filter releasing.Filter
}

func List(opts Opts) error {
	environmentsDir := filepath.Join(opts.BaseDir, "environments")
	environments, err := environment.LoadAll(environmentsDir)
	if err != nil {
		return fmt.Errorf("loading environments: %w", err)
	}

	list, err := releasing.LoadCrossReleaseList(environmentsDir, environments, opts.Filter)
	if err != nil {
		return fmt.Errorf("loading cross-environment releases: %w", err)
	}

	list.Print()
	return nil
}
