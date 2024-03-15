package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/nestoca/joy/internal/helm"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

const (
	CatalogConfigFile = "joy.yaml"
	JoyrcFile         = ".joyrc"
	JoyDefaultDir     = ".joy"
)

type Config struct {
	// CatalogDir is the directory containing catalog of environments, projects and releases.
	// Optional, defaults to ~/.joy
	CatalogDir string `yaml:"catalogDir,omitempty"`

	// Environments user has selected to work with.
	Environments Environments `yaml:"environments,omitempty"`

	// Releases user has selected to work with.
	Releases Releases `yaml:"releases,omitempty"`

	// MinVersion is the minimum version of the joy CLI required
	MinVersion string `yaml:"minVersion,omitempty"`

	// Charts are the known charts that environments and releases can reference
	Charts map[string]helm.Chart `yaml:"charts,omitempty"`

	// DefaultChartRef refers to the chart that must be used from Charts if a release doesn't specify any chart configuration
	DefaultChartRef string `yaml:"defaultChartRef,omitempty"`

	// ReferenceEnvironment is the name of the environment which represents master in git.
	// IE: if you deploy by default to an environment called "testing" when merging to your main remote branch
	// then referenceEnvironment should be "testing". This setting allows release versions to be compared to main version.
	ReferenceEnvironment string `yaml:"referenceEnvironment,omitempty"`

	// ValueMapping are used to apply parameters to the chart values. The values of the mapping
	// can use the Release and Environment as template values. Chart mappings will not override values
	// already present in the chart.
	// For example:
	//
	//   image.tag: {{ .Release.Spec.Version }}
	//   common.annotations.example\.com/custom: true
	//
	ValueMapping *ValueMapping `yaml:"valueMapping,omitempty"`

	// FilePath is the path to the config file that was loaded, used to write back to the same file.
	FilePath string `yaml:"-"`

	JoyCache string `yaml:"-"`

	RepositoriesDir string `yaml:"repositoriesDir,omitempty"`

	// Default GitHub organization to infer the repository from the project name.
	GitHubOrganization string `yaml:"gitHubOrganization,omitempty"`

	Templates Templates `yaml:"templates,omitempty"`
}

type Templates struct {
	Environment EnvironmentTemplates `yaml:"environment,omitempty"`
	Project     ProjectTemplates     `yaml:"project,omitempty"`
	Release     ReleaseTemplates     `yaml:"release,omitempty"`
}

type EnvironmentTemplates struct {
	Links map[string]string `yaml:"links,omitempty"`
}

type ProjectTemplates struct {
	GitTag string            `yaml:"gitTag,omitempty"`
	Links  map[string]string `yaml:"links,omitempty"`
}

type ReleaseTemplates struct {
	Promote ReleasePromoteTemplates `yaml:"promote,omitempty"`
	Links   map[string]string       `yaml:"links,omitempty"`
}

type ReleasePromoteTemplates struct {
	Commit      string `yaml:"commit,omitempty"`
	PullRequest string `yaml:"pullRequest,omitempty"`
}

type ValueMapping struct {
	ReleaseIgnoreList []string
	Mappings          map[string]any
}

// UnmarshalYAML provides custom unmarshalling for backwards compatibility with map[string]string valueMappings.
// This is a stop gap so that we do not break the current joy CLI interpretation of the catalog.
// However, this will enable us to add a releaseIgnoreList to ignore injecting default values into charts
// that would otherwise break.
func (mapping *ValueMapping) UnmarshalYAML(node *yaml.Node) error {
	// Cannot decode directly to mapping otherwise we have entered the infinite recursive look up unmarshalling
	var value struct {
		ReleaseIgnoreList []string       `yaml:"releaseIgnoreList,omitempty"`
		Mappings          map[string]any `yaml:"mappings,omitempty"`
	}

	if err := node.Decode(&value); err == nil && len(value.Mappings) > 0 {
		*mapping = ValueMapping(value)
		return nil
	}

	// for backwards compatibility with versions that declared value mappings as map[string]string
	// we need to be able unmarshal that structure.
	return node.Decode(&mapping.Mappings)
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
	joyrcPath := filepath.Join(configDir, JoyrcFile)

	cfg, err := LoadFile(joyrcPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", joyrcPath, err)
	}

	rootCache := os.Getenv("XDG_CACHE_HOME")
	if rootCache == "" {
		rootCache = filepath.Join(homeDir, ".cache")
	}

	cfg.JoyCache = filepath.Join(rootCache, "joy")

	if catalogDir != "" {
		cfg.CatalogDir = catalogDir
	}

	if cfg.CatalogDir == "" {
		cfg.CatalogDir = filepath.Join(homeDir, JoyDefaultDir)
	}

	catalogConfigPath := filepath.Join(cfg.CatalogDir, CatalogConfigFile)

	catalogCfg, err := LoadFile(catalogConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog configuration: %w", err)
	}

	if cfg.MinVersion != "" && !semver.IsValid(cfg.MinVersion) {
		return nil, fmt.Errorf("invalid minimum version: %s", cfg.MinVersion)
	}

	deepCopyConfigNonZeroValues(catalogCfg, cfg)

	return cfg, nil
}

func deepCopyConfigNonZeroValues(source, target *Config) {
	deepCopyNonZeroValues(reflect.ValueOf(source).Elem(), reflect.ValueOf(target).Elem())
}

func deepCopyNonZeroValues(src, tgt reflect.Value) {
	for i := 0; i < src.NumField(); i++ {
		srcField := src.Field(i)
		tgtField := tgt.Field(i)

		switch srcField.Kind() {
		case reflect.Struct:
			deepCopyNonZeroValues(srcField, tgtField)
		case reflect.Map:
			if srcField.Len() > 0 {
				if tgtField.IsNil() {
					tgtField.Set(reflect.MakeMap(tgtField.Type()))
				}
				for _, key := range srcField.MapKeys() {
					srcValue := srcField.MapIndex(key)
					tgtField.SetMapIndex(key, srcValue)
				}
			}
		default:
			if !isZeroValue(srcField.Interface()) {
				tgtField.Set(srcField)
			}
		}
	}
}

func isZeroValue(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
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
	cfg := &Config{FilePath: file}

	// Load config from file if it exists
	content, err := os.ReadFile(file)
	if err != nil {
		// It's ok if config file does not exist, but not any other errors.
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("loading config %s: %w", file, err)
	}

	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling %s: %w", file, err)
	}

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

type cfgKey struct{}

func ToContext(parent context.Context, cfg *Config) context.Context {
	return context.WithValue(parent, cfgKey{}, cfg)
}

func FromContext(ctx context.Context) *Config {
	cfg, _ := ctx.Value(cfgKey{}).(*Config)
	return cfg
}
