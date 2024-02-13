package joy

import (
	"path/filepath"

	"github.com/nestoca/joy/internal/config"
)

type Config = config.Config

// LoadCatalogConfig takes the path to the catalog as input and loads any catalog specific
// configuration found in its .joyrc
func LoadCatalogConfig(catalogPath string) (*Config, error) {
	return config.LoadFile(filepath.Join(catalogPath, config.JoyrcFile))
}
