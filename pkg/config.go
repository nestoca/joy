package joy

import (
	"context"

	"github.com/nestoca/joy/internal/config"
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
