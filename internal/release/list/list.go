package list

import (
	"fmt"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/pkg/catalog"
)

type Opts struct {
	// CatalogDir is the path to the catalog directory.
	CatalogDir string

	// SelectedEnvs is the list of environments that were selected by user to work with.
	SelectedEnvs []string

	// Filter specifies releases to list.
	// Optional, defaults to listing all releases.
	Filter filtering.Filter
}

func List(opts Opts) error {
	err := git.EnsureCleanAndUpToDateWorkingCopy(opts.CatalogDir)
	if err != nil {
		return err
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		Dir:             opts.CatalogDir,
		LoadEnvs:        true,
		LoadReleases:    true,
		EnvNames:        opts.SelectedEnvs,
		SortEnvsByOrder: true,
		ReleaseFilter:   opts.Filter,
	}
	cat, err := catalog.Load(loadOpts)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	cat.Releases.Print(cross.PrintOpts{})
	return nil
}
