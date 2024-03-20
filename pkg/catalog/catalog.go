package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gopkg.in/godo.v2/glob"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/yml"
)

// Export internal catalog types so that they can worked with from code that use
// the public joy packages.
type (
	ReleaseList = cross.ReleaseList
	Release     = cross.Release
)

type Catalog struct {
	Environments []*v1alpha1.Environment
	Releases     *ReleaseList
	Projects     []*v1alpha1.Project
	Files        []*yml.File
}

// LoadOpts controls how to load catalog and what to load in it.
type LoadOpts struct {
	// Dir is the directory to load catalog from.
	Dir string

	// EnvNames is the list of environment names to load.
	EnvNames []string

	// SortByOrder controls whether environments should be sorted by their spec.order property.
	SortEnvsByOrder bool

	// ReleaseFilter allows to specify which releases to load.
	// Optional, defaults to loading all releases.
	ReleaseFilter filtering.Filter
}

func Load(opts LoadOpts) (*Catalog, error) {
	// Get absolute and clean path of directory, so we can determine whether a release belongs to an environment
	// by simply comparing the beginning of their paths.
	dir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path of %s: %w", dir, err)
	}
	dir = filepath.Clean(dir)

	// Ensure directory exists
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("catalog directory not found: %q", dir)
		}
	}

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

	c.Environments, err = c.loadEnvironments(opts.EnvNames, opts.SortEnvsByOrder)
	if err != nil {
		return nil, fmt.Errorf("loading environments: %w", err)
	}

	c.Projects, err = c.loadProjects()
	if err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}

	allReleaseFiles := c.GetFilesByKind(v1alpha1.ReleaseKind)
	if err := validateTagsForFiles(allReleaseFiles); err != nil {
		return nil, fmt.Errorf("release files with invalid tags: %w", err)
	}

	for _, file := range allReleaseFiles {
		validateTags(file.Tree)
	}

	c.Releases, err = cross.LoadReleaseList(allReleaseFiles, c.Environments, c.Projects, opts.ReleaseFilter)
	if err != nil {
		return nil, fmt.Errorf("loading cross-environment releases: %w", err)
	}

	if err := c.ResolveRefs(); err != nil {
		return nil, fmt.Errorf("resolving references: %w", err)
	}

	return c, nil
}

func (c *Catalog) ResolveRefs() error {
	var errs []error
	if len(c.Releases.Items) > 0 {
		// Resolve references from releases to projects
		if len(c.Projects) > 0 {
			err := c.Releases.ResolveProjectRefs(c.Projects)
			if err != nil {
				errs = append(errs, err)
			}
		}

		// Resolve references from releases to environments
		if len(c.Environments) > 0 {
			c.Releases.ResolveEnvRefs(c.Environments)
		}
	}
	return errors.Join(errs...)
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

func (c *Catalog) loadEnvironments(names []string, sortByOrder bool) ([]*v1alpha1.Environment, error) {
	// Load all environment files
	files := c.GetFilesByKind(v1alpha1.EnvironmentKind)

	// If names is empty, load all environments
	explicitEnvs := len(names) > 0
	var envs []*v1alpha1.Environment
	if explicitEnvs {
		envs = make([]*v1alpha1.Environment, len(names))
	}

	remainingNames := slices.Clone(names)
	currentIndex := 0
	for _, file := range files {
		// Skip if not in names
		envName := file.MetadataName
		if explicitEnvs && !slices.Contains(names, envName) {
			continue
		}

		// Determine environment index
		var index int
		if explicitEnvs {
			// Use index of explicitly requested environment
			index = slices.Index(names, envName)

			// Skip non-requested environments
			if index == -1 {
				continue
			}
		}

		// Load environment
		env, err := v1alpha1.NewEnvironment(file)
		if err != nil {
			return nil, fmt.Errorf("loading environment from %s: %w", file.Path, err)
		}

		// Add environment to list
		if explicitEnvs {
			envs[index] = env
		} else {
			envs = append(envs, env)
		}
		remainingNames = slices.DeleteFunc(remainingNames, func(x string) bool { return x == envName })
		currentIndex++
	}

	// Ensure we found all requested environments
	if len(remainingNames) > 0 {
		return nil, fmt.Errorf("environments not found: %s", strings.Join(remainingNames, ", "))
	}

	// Sort environments by order
	if sortByOrder {
		sort.Slice(envs, func(i, j int) bool {
			return envs[i].Spec.Order < envs[j].Spec.Order
		})
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
