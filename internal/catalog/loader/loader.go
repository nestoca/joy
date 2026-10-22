package loader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/pkg/catalog"
)

//go:generate moq -stub -out ./loader_mock.go . Loader
type Loader interface {
	// LoadAt returns the catalog as it is at given git ref, and a function removing its temporary checkout.
	LoadAt(ctx context.Context, ref string) (*catalog.Catalog, func() error, error)
}

// Load loads the joy config and catalog of the catalog checked out at given dir, which may differ from the configured
// catalog dir, such as a worktree checked out at a given git ref.
func Load(ctx context.Context, dir string) (*config.Config, *catalog.Catalog, error) {
	cfg, err := config.Load(ctx, "", dir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}

	cat, err := catalog.Load(ctx, dir, cfg.KnownChartRefs())
	if err != nil {
		return nil, nil, fmt.Errorf("loading catalog (%s): %w", dir, err)
	}

	return cfg, cat, nil
}

type WorktreeLoader struct {
	dir string
}

// NewWorktreeLoader returns a loader checking out the catalog repository at given dir in temporary worktrees.
func NewWorktreeLoader(dir string) *WorktreeLoader {
	return &WorktreeLoader{dir: dir}
}

// LoadAt loads the catalog in a temporary worktree checked out at given git ref.
func (l *WorktreeLoader) LoadAt(ctx context.Context, ref string) (*catalog.Catalog, func() error, error) {
	sha, err := git.RevParse(l.dir, ref)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving ref %s: %w", ref, err)
	}

	absDir, err := filepath.Abs(l.dir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving catalog dir: %w", err)
	}

	// The worktree keeps the catalog directory name, as release link templates may rely on it (eg: "^.*/catalog/").
	name := filepath.Join("catalog-at-"+sha, filepath.Base(absDir))
	dir, removeWorktree, err := git.CheckoutWorktree(l.dir, name, sha)
	if err != nil {
		return nil, nil, err
	}
	remove := func() error {
		if err := removeWorktree(); err != nil {
			return err
		}
		return os.Remove(filepath.Dir(dir))
	}

	_, cat, err := Load(ctx, dir)
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("loading catalog at %s: %w", ref, err), remove())
	}

	return cat, remove, nil
}
