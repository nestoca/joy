package config

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidmdm/x/xerr"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/pkg/helm"
)

const (
	CatalogConfigFile = "joy.yaml"
	JoyrcFile         = ".joyrc"
	JoyDefaultDir     = ".joy"
)

type User struct {
	// CatalogDir is the directory containing catalog of environments, projects and releases.
	// Optional, defaults to ~/.joy
	CatalogDir string `yaml:"catalogDir,omitempty"`

	// Environments user has selected to work with.
	Environments Environments `yaml:"environments,omitempty"`

	// Releases user has selected to work with.
	Releases Releases `yaml:"releases,omitempty"`

	ColumnWidths ColumnWidths `yaml:"columnWidths,omitempty"`

	// FilePath is the path to the config file that was loaded, used to write back to the same file.
	FilePath string `yaml:"-"`
}

type Catalog struct {
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

	RepositoriesDir string `yaml:"repositoriesDir,omitempty"`

	// Default GitHub organization to infer the repository from the project name.
	GitHubOrganization string `yaml:"gitHubOrganization,omitempty"`

	Templates Templates `yaml:"templates,omitempty"`

	Helps map[string][]Help `yaml:"help,omitempty"`
}

type Config struct {
	User
	Catalog

	JoyCache string
}

const (
	DefaultNarrowColumnWidth = 20
	DefaultNormalColumnWidth = 40
	DefaultWideColumnWidth   = 80
)

type ColumnWidths struct {
	Narrow int `yaml:"narrow,omitempty"`
	Normal int `yaml:"normal,omitempty"`
	Wide   int `yaml:"wide,omitempty"`
}

func (c ColumnWidths) Get(narrow, wide bool) int {
	if narrow {
		return cmp.Or(c.Narrow, DefaultNarrowColumnWidth)
	}
	if wide {
		return cmp.Or(c.Wide, DefaultWideColumnWidth)
	}
	return cmp.Or(c.Normal, DefaultNormalColumnWidth)
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

type Help struct {
	// ErrorPattern is an optional regex pattern to match against the error message to determine if this help message should be displayed.
	ErrorPattern string `yaml:"error,omitempty"`

	// Message is the help message to display.
	Message string `yaml:"message,omitempty"`
}

func (config *Config) KnownChartRefs() []string {
	var refs []string
	for ref := range config.Charts {
		refs = append(refs, ref)
	}
	return refs
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	rootCache := os.Getenv("XDG_CACHE_HOME")
	if rootCache == "" {
		rootCache = filepath.Join(homeDir, ".cache")
	}

	if configDir == "" {
		configDir = homeDir
	}

	joyrcPath := filepath.Join(configDir, JoyrcFile)

	cfg := Config{
		JoyCache: filepath.Join(rootCache, "joy"),
		User: User{
			FilePath: joyrcPath,
		},
	}

	if err := LoadFile(joyrcPath, &cfg.User); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading user config %s: %w", joyrcPath, err)
	}

	cfg.User.CatalogDir = cmp.Or(catalogDir, cfg.User.CatalogDir, filepath.Join(homeDir, JoyDefaultDir))

	catalogConfigPath := filepath.Join(cfg.User.CatalogDir, CatalogConfigFile)

	if err := LoadFile(catalogConfigPath, &cfg.Catalog); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading catalog config %s: %w", catalogConfigPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (cfg Config) Validate() error {
	var errs []error
	for ref, chart := range cfg.Charts {
		if chart.RepoURL == "" || chart.Name == "" || chart.Version == "" {
			errs = append(errs, fmt.Errorf("%s: %s", ref, "chart must be fully qualified: repoUrl, name, and version are required"))
		}
	}
	if err := xerr.MultiErrOrderedFrom("validating charts", errs...); err != nil {
		return err
	}

	if cfg.MinVersion != "" && !semver.IsValid(cfg.MinVersion) {
		return fmt.Errorf("invalid minimum version: %s", cfg.MinVersion)
	}

	return nil
}

// TODO: maybe this function belongs in package catalog?
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

func LoadFile[T any](file string, value *T) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, &value)
}

// Save saves config back to its original file.
func (user User) Save() error {
	// Marshal config to YAML
	content, err := yaml.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(user.FilePath, content, 0o644); err != nil {
		return fmt.Errorf("writing config to %s: %w", user.FilePath, err)
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
