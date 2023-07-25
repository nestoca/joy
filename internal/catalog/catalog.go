package catalog

import (
	"fmt"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/project"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/utils"
	"github.com/nestoca/joy/internal/yml"
	"gopkg.in/godo.v2/glob"
	"path/filepath"
	"sort"
)

type Catalog struct {
	Environments  []*environment.Environment
	CrossReleases *release.CrossReleaseList
	Projects      []*project.Project
	Files         []*yml.File
}

type LoadOpts struct {
	// Dir is the directory to load catalog from.
	Dir string

	// EnvNames is the list of environment names to load.
	EnvNames []string

	// SortEnvsByOrder controls whether environments should be sorted by their spec.order property.
	SortEnvsByOrder bool

	// ReleaseFilter allows to specify which releases to load.
	// Optional, defaults to loading all releases.
	ReleaseFilter release.Filter
}

func Load(opts LoadOpts) (*Catalog, error) {
	// Get absolute and clean path of directory, so we can determine whether a release belongs to an environment
	// by simply comparing the beginning of their paths.
	dir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path of %s: %w", dir, err)
	}
	dir = filepath.Clean(dir)

	// Find all files matching the glob expression
	globExpr := filepath.Join(dir, "**/*.yaml")
	fileAssets, _, err := glob.Glob([]string{globExpr})
	if err != nil {
		return nil, fmt.Errorf("matching files with glob expression %s: %w", globExpr, err)
	}

	// Load all matching files
	c := &Catalog{}
	for _, fileAsset := range fileAssets {
		file, err := yml.LoadFile(fileAsset.Path)
		if err != nil {
			return nil, fmt.Errorf("loading yaml file %s: %w", fileAsset.Path, err)
		}
		// Only keep Joy CRDs
		if isValid(file) {
			c.Files = append(c.Files, file)
		}
	}

	// Load environments
	c.Environments, err = c.loadEnvironments(opts.EnvNames...)
	if err != nil {
		return nil, fmt.Errorf("loading environments: %w", err)
	}

	// Sort environments by order
	if opts.SortEnvsByOrder {
		sort.Slice(c.Environments, func(i, j int) bool {
			return c.Environments[i].Spec.Order < c.Environments[j].Spec.Order
		})
	}

	// Load projects
	c.Projects, err = c.loadProjects()
	if err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}

	// Load cross-releases
	allReleaseFiles := c.GetFilesByKind(release.Kind)
	c.CrossReleases, err = release.LoadCrossReleaseList(allReleaseFiles, c.Environments, opts.ReleaseFilter)
	if err != nil {
		return nil, fmt.Errorf("loading cross-environment releases: %w", err)
	}

	return c, nil
}

func isValid(file *yml.File) bool {
	return environment.IsValid(file.ApiVersion, file.Kind) ||
		release.IsValid(file.ApiVersion, file.Kind) ||
		project.IsValid(file.ApiVersion, file.Kind)
}

// GetFilesByKind returns all files of the given kind.
func (c *Catalog) GetFilesByKind(kind string) []*yml.File {
	var files []*yml.File
	for _, file := range c.Files {
		if file.Kind == kind {
			files = append(files, file)
		}
	}
	return files
}

func (c *Catalog) loadEnvironments(names ...string) ([]*environment.Environment, error) {
	files := c.GetFilesByKind(environment.Kind)

	var envs []*environment.Environment
	for _, file := range files {
		// Skip if not in names
		if len(names) > 0 && !utils.SliceContainsString(names, file.MetadataName) {
			continue
		}

		// Load environment
		env, err := environment.New(file)
		if err != nil {
			return nil, fmt.Errorf("loading environment from %s: %w", file.Path, err)
		}
		envs = append(envs, env)
	}

	return envs, nil
}

func (c *Catalog) loadProjects() ([]*project.Project, error) {
	files := c.GetFilesByKind(project.Kind)

	var projects []*project.Project
	for _, file := range files {
		proj, err := project.New(file)
		if err != nil {
			return nil, fmt.Errorf("loading project from %s: %w", file.Path, err)
		}
		projects = append(projects, proj)
	}

	// Sort projects by name
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}
