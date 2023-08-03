package catalog

import (
	"fmt"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/yml"
	"golang.org/x/exp/slices"
	"gopkg.in/godo.v2/glob"
	"path/filepath"
	"sort"
)

type Catalog struct {
	Environments []*v1alpha1.Environment
	Releases     *cross.ReleaseList
	Projects     []*v1alpha1.Project
	Files        []*yml.File
}

// LoadOpts controls how to load catalog and what to load in it.
type LoadOpts struct {
	// Dir is the directory to load catalog from.
	Dir string

	// LoadEnvs controls whether to load environments.
	LoadEnvs bool

	// EnvNames is the list of environment names to load.
	EnvNames []string

	// SortByOrder controls whether environments should be sorted by their spec.order property.
	SortEnvsByOrder bool

	// LoadReleases controls whether to load releases.
	LoadReleases bool

	// ReleaseFilter allows to specify which releases to load.
	// Optional, defaults to loading all releases.
	ReleaseFilter filtering.Filter

	// LoadProjects controls whether to load projects.
	LoadProjects bool

	// ResolveRefs controls whether to resolve references to related resources. Requires that all referenced resources
	// are loaded in the catalog.
	//
	// For example, if ResolveRefs, LoadReleases and LoadEnvs are all enabled, the release.Environment field will be resolved to the
	// actual environment object.
	ResolveRefs bool
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

	if opts.LoadEnvs {
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
	}

	// Load projects
	if opts.LoadProjects {
		c.Projects, err = c.loadProjects()
		if err != nil {
			return nil, fmt.Errorf("loading projects: %w", err)
		}
	}

	// Load cross-releases
	if opts.LoadReleases {
		allReleaseFiles := c.GetFilesByKind(v1alpha1.ReleaseKind)
		c.Releases, err = cross.LoadReleaseList(allReleaseFiles, c.Environments, opts.ReleaseFilter)
		if err != nil {
			return nil, fmt.Errorf("loading cross-environment releases: %w", err)
		}
	}

	// Resolve references
	if opts.ResolveRefs && opts.LoadReleases {
		// Resolve references from releases to projects
		if opts.LoadProjects {
			err = c.Releases.ResolveProjectRefs(c.Projects)
			if err != nil {
				return nil, fmt.Errorf("resolving project references: %w", err)
			}
		}

		// Resolve references from releases to environments
		if opts.LoadEnvs {
			err = c.Releases.ResolveEnvRefs(c.Environments)
			if err != nil {
				return nil, fmt.Errorf("resolving environment references: %w", err)
			}
		}
	}

	return c, nil
}

func isValid(file *yml.File) bool {
	return v1alpha1.IsValidEnvironment(file.ApiVersion, file.Kind) ||
		v1alpha1.IsValidRelease(file.ApiVersion, file.Kind) ||
		v1alpha1.IsValidProject(file.ApiVersion, file.Kind)
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

func (c *Catalog) loadEnvironments(names ...string) ([]*v1alpha1.Environment, error) {
	files := c.GetFilesByKind(v1alpha1.EnvironmentKind)

	var envs []*v1alpha1.Environment
	for _, file := range files {
		// Skip if not in names
		if len(names) > 0 && !slices.Contains(names, file.MetadataName) {
			continue
		}

		// Load environment
		env, err := v1alpha1.NewEnvironment(file)
		if err != nil {
			return nil, fmt.Errorf("loading environment from %s: %w", file.Path, err)
		}
		envs = append(envs, env)
	}

	return envs, nil
}

func (c *Catalog) loadProjects() ([]*v1alpha1.Project, error) {
	files := c.GetFilesByKind(v1alpha1.ProjectKind)

	var projects []*v1alpha1.Project
	for _, file := range files {
		proj, err := v1alpha1.NewProject(file)
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
