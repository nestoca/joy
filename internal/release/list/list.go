package list

import (
	"fmt"
	"github.com/nestoca/joy/internal/catalog"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/release"
)

type Opts struct {
	// SelectedEnvs is the list of environments that were selected by user to work with.
	SelectedEnvs []string

	// Filter specifies releases to list.
	// Optional, defaults to listing all releases.
	Filter release.Filter
}

func List(opts Opts) error {
	err := git.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return err
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		EnvNames:        opts.SelectedEnvs,
		SortEnvsByOrder: true,
		ReleaseFilter:   opts.Filter,
	}
	cat, err := catalog.Load(loadOpts)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	cat.CrossReleases.Print(release.PrintOpts{})
	return nil
}
