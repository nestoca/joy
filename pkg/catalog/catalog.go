package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/davidmdm/x/xerr"
	"gopkg.in/godo.v2/glob"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/ignore"
	"github.com/nestoca/joy/internal/observability"
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
	Releases     ReleaseList
	Projects     []*v1alpha1.Project
	Files        []*yml.File
}

func Load(ctx context.Context, dir string, validChartRefs []string) (*Catalog, error) {
	_, span := observability.StartTrace(ctx, "load_catalog")
	defer span.End()

	// Get absolute and clean path of directory, so we can determine whether a release belongs to an environment
	// by simply comparing the beginning of their paths.
	dir, err := filepath.Abs(dir)
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

	// Load .joyignore if it exists
	ignoreMatcher, err := ignore.NewMatcher(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading .joyignore: %w", err)
	}

	// Load all matching files
	c := &Catalog{}
	for _, fileAsset := range fileAssets {
		if ignoreMatcher != nil && ignoreMatcher.Match(fileAsset.Path) {
			continue
		}

		file, err := yml.LoadFile(fileAsset.Path)
		if err != nil {
			return nil, fmt.Errorf("loading yaml file %s: %w", fileAsset.Path, err)
		}
		// Only keep Joy CRDs
		if isValid(file) {
			c.Files = append(c.Files, file)
		}
	}

	c.Environments, err = c.loadEnvironments(nil, true)
	if err != nil {
		return nil, fmt.Errorf("loading environments: %w", err)
	}

	var errs []error
	for _, env := range c.Environments {
		if err := env.Validate(validChartRefs); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", env.Name, err))
		}
	}

	if err := xerr.MultiErrOrderedFrom("validating environments", errs...); err != nil {
		return nil, err
	}

	c.Projects, err = c.loadProjects()
	if err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}

	for _, project := range c.Projects {
		if err := project.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", project.Name, err))
		}
	}

	if err := xerr.MultiErrOrderedFrom("validating project", errs...); err != nil {
		return nil, err
	}

	allReleaseFiles := c.GetFilesByKind(v1alpha1.ReleaseKind)

	if err := validateTagsForFiles(allReleaseFiles); err != nil {
		return nil, fmt.Errorf("release files with invalid tags: %w", err)
	}

	c.Releases, err = cross.LoadReleaseList(allReleaseFiles, c.Environments, c.Projects, nil)
	if err != nil {
		return nil, fmt.Errorf("loading cross-environment releases: %w", err)
	}

	if err := c.ResolveRefs(); err != nil {
		return nil, fmt.Errorf("resolving references: %w", err)
	}

	for _, cross := range c.Releases.Items {
		for _, release := range cross.Releases {
			if release == nil {
				continue
			}
			if err := release.Spec.Chart.Validate(validChartRefs); err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: invalid chart: %w", release.Name, release.Environment.Name, err))
			}
			if err := release.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: validation: %w", release.Name, release.Environment.Name, err))
			}
		}
	}
	if err := xerr.MultiErrOrderedFrom("validating releases", errs...); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Catalog) WithReleaseFilter(filter filtering.Filter) {
	if filter == nil {
		return
	}

	releases := c.Releases.Items
	c.Releases.Items = []*cross.Release{}

	for _, cross := range releases {
		for i, rel := range cross.Releases {
			if rel == nil || !filter.Match(rel) {
				cross.Releases[i] = nil
			}
		}
		if !slices.ContainsFunc(cross.Releases, func(rel *v1alpha1.Release) bool { return rel != nil }) {
			continue
		}
		c.Releases.Items = append(c.Releases.Items, cross)
	}
}

func (c *Catalog) WithReleases(names []string) {
	if len(names) == 0 {
		return
	}

	releases := c.Releases.Items
	c.Releases.Items = []*cross.Release{}

	for _, cross := range releases {
		if slices.Contains(names, cross.Name) {
			c.Releases.Items = append(c.Releases.Items, cross)
		}
	}
}

func (c *Catalog) WithEnvironments(names []string) {
	if len(names) == 0 {
		return
	}

	allEnvs := c.Environments
	c.Environments = []*v1alpha1.Environment{}

	var removedIndexes []int
	for i, env := range allEnvs {
		if slices.Contains(names, env.Name) {
			c.Environments = append(c.Environments, env)
		} else {
			removedIndexes = append(removedIndexes, i)
		}
	}

	c.Releases.Environments = c.Environments

	for _, cross := range c.Releases.Items {
		releases := cross.Releases
		cross.Releases = []*v1alpha1.Release{}
		for j, rel := range releases {
			if !slices.Contains(removedIndexes, j) {
				cross.Releases = append(cross.Releases, rel)
			}
		}
	}
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

func (c *Catalog) GetReleaseNames() []string {
	result := []string{}
	for _, cross := range c.Releases.Items {
		if slices.ContainsFunc(cross.Releases, func(release *v1alpha1.Release) bool {
			return release != nil
		}) {
			result = append(result, cross.Name)
		}
	}
	return result
}

func (c *Catalog) GetEnvironmentNames() []string {
	result := []string{}
	for _, env := range c.Environments {
		result = append(result, env.Name)
	}
	return result
}

func (c *Catalog) LookupRelease(env, release string) (*v1alpha1.Release, error) {
	if !slices.Contains(c.GetEnvironmentNames(), env) {
		return nil, fmt.Errorf("unknown environment: %s", env)
	}
	if !slices.Contains(c.GetReleaseNames(), release) {
		return nil, fmt.Errorf("unknown release: %s", release)
	}

	for _, cross := range c.Releases.Items {
		if cross.Name != release {
			continue
		}
		for _, rel := range cross.Releases {
			if rel == nil {
				continue
			}
			if rel.Environment.Name == env {
				return rel, nil
			}
		}
		break
	}

	return nil, fmt.Errorf("release %s not found in environment %s", release, env)
}

type catalogKey struct{}

func ToContext(ctx context.Context, catalog *Catalog) context.Context {
	return context.WithValue(ctx, catalogKey{}, catalog)
}

func FromContext(ctx context.Context) *Catalog {
	catalog, _ := ctx.Value(catalogKey{}).(*Catalog)
	if catalog == nil {
		panic("catalog loaded from context but is absent: cannot continue")
	}
	return catalog
}
