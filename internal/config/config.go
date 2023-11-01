package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	joyrcFile     = ".joyrc"
	joyDefaultDir = ".joy"
)

type Config struct {
	// CatalogDir is the directory containing catalog of environments, projects and releases.
	// Optional, defaults to ~/.joy
	CatalogDir string `yaml:"catalogDir,omitempty"`

	// Environments user has selected to work with.
	Environments Environments `yaml:"environments,omitempty"`

	// Releases user has selected to work with.
	Releases Releases `yaml:"releases,omitempty"`

	// FilePath is the path to the config file that was loaded, used to write back to the same file.
	FilePath string `yaml:"-"`
}

type Environments struct {
	// Selected is the list of environments user has selected to work with.
	// Only those will be displayed in table columns by default.
	// An empty list means all environments are selected.
	Selected []string `yaml:"selected,omitempty"`
}

type Releases struct {
	// Selected is the list of releases user has selected to work with.
	// Only those will be displayed in table rows by default.
	// An empty list means all releases are selected.
	Selected []string `yaml:"selected,omitempty"`
}

// Load loads config from given configDir (or user home if not specified) and
// optionally overrides loaded config's catalog directory with given catalogDir,
// defaulting to ~/.joy if not specified.
func Load(configDir, catalogDir string) (*Config, error) {
	// Default configDir to user home
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	if configDir == "" {
		configDir = homeDir
	}

	// Load config from .joyrc in configDir
	var cfg *Config
	joyrcPath := filepath.Join(configDir, joyrcFile)
	cfg, err = LoadFile(joyrcPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", joyrcPath, err)
	}

	// Set defaults and in-memory values
	if catalogDir != "" {
		cfg.CatalogDir = catalogDir
	} else if cfg.CatalogDir == "" {
		cfg.CatalogDir = filepath.Join(homeDir, joyDefaultDir)
	}
	cfg.FilePath = joyrcPath

	return cfg, nil
}

func CheckCatalogDir(catalogDir string) error {
	// Ensure that catalog's environments directory exists
	environmentsDir := filepath.Join(catalogDir, "environments")
	if _, err := os.Stat(environmentsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no joy catalog found at %q", catalogDir)
		}
		return fmt.Errorf("checking for catalog directory %s: %w", catalogDir, err)
	}
	return nil
}

func LoadFile(file string) (*Config, error) {
	cfg := &Config{}

	_, err := os.Stat(file)
	if err == nil {
		// Load config from file if it exists
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("loading config %s: %w", file, err)
		}
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshalling %s: %w", file, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		// It's ok if config file does not exist, but not any other errors.
		return nil, fmt.Errorf("checking for config file %s: %w", file, err)
	}

	// Saving file location
	cfg.FilePath = file
	return cfg, nil
}

// Save saves config back to its original file.
func (c *Config) Save() error {
	// Marshal config to YAML
	content, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.FilePath, content, 0o644); err != nil {
		return fmt.Errorf("writing config to %s: %w", c.FilePath, err)
	}

	return nil
}
