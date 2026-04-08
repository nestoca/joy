package joy

import (
	"context"
	"fmt"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/pkg/catalog"
)

type (
	Config        = config.Config
	UserConfig    = config.User
	CatalogConfig = config.Catalog
)

// LoadConfigFromCatalog takes the path to the catalog as input and loads any catalog specific
// configuration found in its .joyrc
func LoadConfigFromCatalog(ctx context.Context, catalogPath string) (*Config, error) {
	return config.Load(ctx, "", catalogPath)
}

// LoadCatalog loads the catalog from a given path
func LoadCatalog(ctx context.Context, path string) (*catalog.Catalog, error) {
	cfg, err := LoadConfigFromCatalog(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}
	return catalog.Load(ctx, cfg.CatalogDir, cfg.KnownChartRefs())
}
