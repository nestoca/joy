package joy

import (
	"github.com/nestoca/joy/internal/config"
)

type (
	Config        = config.Config
	UserConfig    = config.User
	CatalogConfig = config.Catalog
)

// LoadConfigFromCatalog takes the path to the catalog as input and loads any catalog specific
// configuration found in its .joyrc
func LoadConfigFromCatalog(catalogPath string) (*Config, error) {
	var cfg Config
	if err := config.LoadFile(catalogPath, &cfg.Catalog); err != nil {
		return nil, err
	}
	return &cfg, nil
}
