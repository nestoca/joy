package cross

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/davidmdm/x/xerr"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/references"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/yml"
)

// ReleaseList describes multiple releases across multiple environments
type ReleaseList struct {
	Environments []*v1alpha1.Environment
	Items        []*Release
}

// MakeReleaseList creates a new ReleaseList
func MakeReleaseList(environments []*v1alpha1.Environment) ReleaseList {
	return ReleaseList{
		Environments: environments,
	}
}

// LoadReleaseList loads all releases for given environments underneath the given base directory.
func LoadReleaseList(allFiles []*yml.File, environments []*v1alpha1.Environment, projects []*v1alpha1.Project, releaseFilter filtering.Filter) (ReleaseList, error) {
	crossReleases := MakeReleaseList(environments)
	for _, file := range allFiles {
		env := findEnvironmentForReleaseFile(environments, file)
		if env == nil {
			continue
		}

		rel, err := v1alpha1.LoadRelease(file)
		if err != nil {
			return ReleaseList{}, fmt.Errorf("loading release %s: %w", file.Path, err)
		}

		if rel.Spec.Project != "" && projects != nil && len(projects) != 0 {
			proj := findProjectForRelease(projects, rel)
			if proj != nil {
				rel.Project = proj
			}
		}

		// Filter out releases that don't match the filter
		if releaseFilter != nil && !releaseFilter.Match(rel) {
			continue
		}

		// Add release to cross-release list
		err = crossReleases.addRelease(rel, env)
		if err != nil {
			return ReleaseList{}, fmt.Errorf("adding release %s to environment %s: %w", rel.Name, env.Name, err)
		}
	}

	// Sort cross-releases by name
	sort.Slice(crossReleases.Items, func(i, j int) bool {
		return crossReleases.Items[i].Name < crossReleases.Items[j].Name
	})

	return crossReleases, nil
}

// findEnvironmentForReleaseFile returns the environment that contains the given release file.
// Release files are assumed to be within the same directory (or any recursive subdirectory) as the environment file.
func findEnvironmentForReleaseFile(environments []*v1alpha1.Environment, releaseFile *yml.File) *v1alpha1.Environment {
	for _, env := range environments {
		if strings.HasPrefix(releaseFile.Path, env.Dir) {
			return env
		}
	}
	return nil
}

// GetEnvironmentIndexByName returns the index of the environment with the given name or -1 if not found.
func (r *ReleaseList) GetEnvironmentIndexByName(name string) int {
	for i, env := range r.Environments {
		if env.Name == name {
			return i
		}
	}
	return -1
}

func (r *ReleaseList) GetEnvironmentIndex(environment *v1alpha1.Environment) int {
	for i, env := range r.Environments {
		if env == environment {
			return i
		}
	}
	return -1
}

// getReleaseIndex returns the index of the release with the given name or -1 if not found.
func (r *ReleaseList) getReleaseIndex(name string) int {
	for i, rel := range r.Items {
		if rel.Name == name {
			return i
		}
	}
	return -1
}

// addRelease adds a release to given environment.
func (r *ReleaseList) addRelease(rel *v1alpha1.Release, environment *v1alpha1.Environment) error {
	// Find environment index
	environmentIndex := r.GetEnvironmentIndexByName(environment.Name)
	if environmentIndex == -1 {
		return fmt.Errorf("environment %s not found in list", environment.Name)
	}

	// Find or create cross-release
	releaseIndex := r.getReleaseIndex(rel.Name)
	var crossRelease *Release
	if releaseIndex != -1 {
		crossRelease = r.Items[releaseIndex]
	} else {
		crossRelease = NewRelease(rel.Name, r.Environments)
		r.Items = append(r.Items, crossRelease)
	}

	// Add release to environment
	crossRelease.Releases[environmentIndex] = rel
	return nil
}

func (r *ReleaseList) SortedCrossReleases() []*Release {
	releases := append([]*Release{}, r.Items...)
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Name < releases[j].Name
	})
	return releases
}

func (r *ReleaseList) Filter(fn func(release *Release) bool) ReleaseList {
	subset := MakeReleaseList(r.Environments)
	for _, item := range r.Items {
		if fn(item) {
			subset.Items = append(subset.Items, item)
		}
	}
	return subset
}

func (r *ReleaseList) expectReleases(releases []string) error {
	var errs []error
	for _, release := range releases {
		if r.getReleaseIndex(release) == -1 {
			errs = append(errs, errors.New(release))
		}
	}
	return xerr.MultiErrOrderedFrom("release(s) not found", errs...)
}

// OnlySpecificReleases returns a subset of the releases in this list that match the given names.
func (r *ReleaseList) OnlySpecificReleases(releases []string) (ReleaseList, error) {
	if err := r.expectReleases(releases); err != nil {
		return ReleaseList{}, err
	}

	cond := func(release *Release) bool {
		return slices.Contains(releases, release.Name)
	}

	return r.Filter(cond), nil
}

func (r *ReleaseList) RemoveReleasesByName(releases []string) (ReleaseList, error) {
	if err := r.expectReleases(releases); err != nil {
		return ReleaseList{}, err
	}

	cond := func(release *Release) bool {
		return !slices.Contains(releases, release.Name)
	}

	return r.Filter(cond), nil
}

// GetReleasesForPromotion returns a subset of the releases in this list that are promotable,
// with only the given source and target environments as first and second environments.
func (r *ReleaseList) GetReleasesForPromotion(sourceEnv, targetEnv *v1alpha1.Environment) (ReleaseList, error) {
	sourceEnvIndex := r.GetEnvironmentIndexByName(sourceEnv.Name)
	targetEnvIndex := r.GetEnvironmentIndexByName(targetEnv.Name)
	subset := MakeReleaseList([]*v1alpha1.Environment{sourceEnv, targetEnv})
	for _, item := range r.Items {
		// Determine source and target releases
		sourceRelease := item.Releases[sourceEnvIndex]
		targetRelease := item.Releases[targetEnvIndex]

		newItem := NewRelease(item.Name, []*v1alpha1.Environment{sourceEnv, targetEnv})
		newItem.Releases = []*v1alpha1.Release{sourceRelease, targetRelease}

		// Compute promoted file
		err := newItem.ComputePromotedFile(sourceEnv, targetEnv)
		if err != nil {
			return ReleaseList{}, fmt.Errorf("computing promoted file for release %s: %w", item.Name, err)
		}
		subset.Items = append(subset.Items, newItem)
	}
	return subset, nil
}

// GetNonPromotableReleases returns a list of names of releases that cannot be promoted based
// on the version format allowed at the target environment.
func (r *ReleaseList) GetNonPromotableReleases(sourceEnv, targetEnv *v1alpha1.Environment) []string {
	if targetEnv.Spec.Promotion.FromPullRequests {
		return nil
	}

	var invalidList []string
	sourceEnvIndex := r.GetEnvironmentIndexByName(sourceEnv.Name)

	for _, item := range r.Items {
		sourceRelease := item.Releases[sourceEnvIndex]
		if sourceRelease == nil {
			continue
		}
		// Check the version format in source Release
		version := "v" + sourceRelease.Spec.Version
		if semver.Prerelease(version)+semver.Build(version) != "" {
			invalidList = append(invalidList, item.Name)
		}
	}
	return invalidList
}

func (r *ReleaseList) ResolveProjectRefs(projects []*v1alpha1.Project) error {
	var errs []error
	for _, crossRelease := range r.Items {
		for _, rel := range crossRelease.Releases {
			if rel == nil || rel.Spec.Project == "" {
				continue
			}
			proj := findProjectForRelease(projects, rel)
			if proj == nil {
				errs = append(errs, references.NewMissingError("Release", rel.Name, "Project", rel.Spec.Project))
			}
			rel.Project = proj
		}
	}
	return errors.Join(errs...)
}

func (r *ReleaseList) ResolveEnvRefs(environments []*v1alpha1.Environment) {
	for _, crossRelease := range r.Items {
		for i, rel := range crossRelease.Releases {
			if rel != nil {
				rel.Environment = environments[i]
			}
		}
	}
}

func findProjectForRelease(projects []*v1alpha1.Project, rel *v1alpha1.Release) *v1alpha1.Project {
	for _, proj := range projects {
		if proj.Name == rel.Spec.Project {
			return proj
		}
	}
	return nil
}

func (r *ReleaseList) HasAnyPromotableReleases() bool {
	for _, crossRelease := range r.Items {
		if crossRelease.PromotedFile != nil {
			return true
		}
	}
	return false
}

func (r *ReleaseList) GetEnvironmentRelease(environment *v1alpha1.Environment, releaseName string) (*v1alpha1.Release, error) {
	releaseIndex := r.getReleaseIndex(releaseName)
	if releaseIndex == -1 {
		return nil, fmt.Errorf("release %s not found in environment %s", releaseName, environment.Name)
	}
	envIndex := r.GetEnvironmentIndex(environment)
	return r.Items[releaseIndex].Releases[envIndex], nil
}
