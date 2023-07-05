package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const joyrcFile = ".joyrc"
const joyDefaultDir = ".joy"

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

	// Source is the environment user is promoting from by default.
	Source string `yaml:"source,omitempty"`

	// Target is the environment user is promoting to by default.
	Target string `yaml:"target,omitempty"`
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
	_, err = os.Stat(joyrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			// It's ok if config file does not exist, we'll use default values
			cfg = &Config{}
		} else {
			return nil, fmt.Errorf("checking for %s: %w", joyrcPath, err)
		}
	} else {
		cfg, err = LoadFile(joyrcPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", joyrcPath, err)
		}
	}

	// Set defaults and in-memory values
	if catalogDir != "" {
		cfg.CatalogDir = catalogDir
	} else if cfg.CatalogDir == "" {
		cfg.CatalogDir = filepath.Join(homeDir, joyDefaultDir)
	}
	cfg.FilePath = joyrcPath

	// Ensure that catalog's environments directory exists
	environmentsDir := filepath.Join(cfg.CatalogDir, "environments")
	if _, err := os.Stat(environmentsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no joy catalog found at %q", cfg.CatalogDir)
		}
		return nil, fmt.Errorf("checking for catalog directory %s: %w", cfg.CatalogDir, err)
	}

	return cfg, nil
}

func LoadFile(file string) (*Config, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("loading config %s: %w", file, err)
	}
	var cfg *Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling %s: %w", file, err)
	}
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
	if err := os.WriteFile(c.FilePath, content, 0644); err != nil {
		return fmt.Errorf("writing config to %s: %w", c.FilePath, err)
	}

	return nil
}
