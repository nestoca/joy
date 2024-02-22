package cross

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/nestoca/joy/internal/references"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/yml"

	"golang.org/x/mod/semver"
)

// ReleaseList describes multiple releases across multiple environments
type ReleaseList struct {
	Environments []*v1alpha1.Environment
	Items        []*Release
}

// NewReleaseList creates a new ReleaseList
func NewReleaseList(environments []*v1alpha1.Environment) *ReleaseList {
	return &ReleaseList{
		Environments: environments,
	}
}

// LoadReleaseList loads all releases for given environments underneath the given base directory.
func LoadReleaseList(allFiles []*yml.File, environments []*v1alpha1.Environment, releaseFilter filtering.Filter) (*ReleaseList, error) {
	crossReleases := NewReleaseList(environments)
	for _, file := range allFiles {
		env := findEnvironmentForReleaseFile(environments, file)
		if env == nil {
			continue
		}

		rel, err := v1alpha1.LoadRelease(file)
		if err != nil {
			return nil, fmt.Errorf("loading release %s: %w", file.Path, err)
		}

		// Filter out releases that don't match the filter
		if releaseFilter != nil && !releaseFilter.Match(rel) {
			continue
		}

		// Add release to cross-release list
		err = crossReleases.addRelease(rel, env)
		if err != nil {
			return nil, fmt.Errorf("adding release %s to environment %s: %w", rel.Name, env.Name, err)
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

// getEnvironmentIndex returns the index of the environment with the given name or -1 if not found.
func (r *ReleaseList) getEnvironmentIndex(name string) int {
	for i, env := range r.Environments {
		if env.Name == name {
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
	environmentIndex := r.getEnvironmentIndex(environment.Name)
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
	var releases []*Release
	for _, rel := range r.Items {
		releases = append(releases, rel)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Name < releases[j].Name
	})
	return releases
}

// OnlySpecificReleases returns a subset of the releases in this list that match the given names.
func (r *ReleaseList) OnlySpecificReleases(releases []string) *ReleaseList {
	subset := NewReleaseList(r.Environments)
	for _, item := range r.Items {
		if slices.Contains(releases, item.Name) {
			subset.Items = append(subset.Items, item)
		}
	}
	return subset
}

// GetReleasesForPromotion returns a subset of the releases in this list that are promotable,
// with only the given source and target environments as first and second environments.
func (r *ReleaseList) GetReleasesForPromotion(sourceEnv, targetEnv *v1alpha1.Environment) (*ReleaseList, error) {
	sourceEnvIndex := r.getEnvironmentIndex(sourceEnv.Name)
	targetEnvIndex := r.getEnvironmentIndex(targetEnv.Name)
	subset := NewReleaseList([]*v1alpha1.Environment{sourceEnv, targetEnv})
	for _, item := range r.Items {
		// Determine source and target releases
		sourceRelease := item.Releases[sourceEnvIndex]
		targetRelease := item.Releases[targetEnvIndex]

		//Check for version in source Release
		if targetEnv.Name == "qa" || targetEnv.Name == "production" {
			version := "v" + sourceRelease.Spec.Version
			if semver.Prerelease(version) != "" || semver.Build(version) != "" {
				continue
			}
		}

		newItem := NewRelease(item.Name, []*v1alpha1.Environment{sourceEnv, targetEnv})
		newItem.Releases = []*v1alpha1.Release{sourceRelease, targetRelease}

		// Compute promoted file
		err := newItem.ComputePromotedFile(sourceEnv, targetEnv)
		if err != nil {
			return nil, fmt.Errorf("computing promoted file for release %s: %w", item.Name, err)
		}
		subset.Items = append(subset.Items, newItem)
	}
	return subset, nil
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
