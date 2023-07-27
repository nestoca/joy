package cross

import (
	"fmt"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/exp/slices"
	"os"
	"sort"
	"strings"
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

// OnlyPromotableReleases returns a subset of the releases in this list that are promotable.
// This assumes that there are two and only two environments and that first one is the source
// and the second one is the target.
func (r *ReleaseList) OnlyPromotableReleases() *ReleaseList {
	subset := NewReleaseList(r.Environments)
	for _, item := range r.Items {
		if item.Promotable() {
			subset.Items = append(subset.Items, item)
		}
	}
	return subset
}

type PrintOpts struct {
	// IsPromoting allows to dim releases that are not promotable (i.e. have no release in first environment)
	IsPromoting bool
}

// Print displays all releases versions across environments in a table format.
func (r *ReleaseList) Print(opts PrintOpts) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetAutoWrapText(true)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetCenterSeparator("")

	headers := []string{"NAME"}
	for _, env := range r.Environments {
		headers = append(headers, strings.ToUpper(env.Name))
	}
	table.SetHeader(headers)

	for _, crossRelease := range r.Items {
		// Check if releases and their values are synced across all environments
		releasesSynced := crossRelease.AllReleasesSynced()
		dimmed := opts.IsPromoting && !crossRelease.Promotable()

		row := []string{stylize(crossRelease.Name, releasesSynced, dimmed)}
		for _, rel := range crossRelease.Releases {
			text := "-"
			if rel != nil && !rel.Missing {
				text = rel.Spec.Version
			}
			text = stylize(text, releasesSynced, dimmed)
			row = append(row, text)
		}
		table.Append(row)
	}

	table.Render()
}

func (r *ReleaseList) ResolveProjectRefs(projects []*v1alpha1.Project) error {
	for _, crossRelease := range r.Items {
		for _, rel := range crossRelease.Releases {
			if rel == nil || rel.Spec.Project == "" {
				continue
			}
			proj := findProjectForRelease(projects, rel)
			if proj == nil {
				return fmt.Errorf("project %s not found for release %s", rel.Spec.Project, rel.Name)
			}
			rel.Project = proj
		}
	}
	return nil
}

func (r *ReleaseList) ResolveEnvRefs(environments []*v1alpha1.Environment) error {
	for _, crossRelease := range r.Items {
		for i, rel := range crossRelease.Releases {
			if rel != nil {
				rel.Environment = environments[i]
			}
		}
	}
	return nil
}

func findProjectForRelease(projects []*v1alpha1.Project, rel *v1alpha1.Release) *v1alpha1.Project {
	for _, proj := range projects {
		if proj.Name == rel.Spec.Project {
			return proj
		}
	}
	return nil
}

func stylize(text string, releasesSynced, dimmed bool) string {
	if dimmed {
		return style.SecondaryInfo(text)
	}
	if !releasesSynced {
		return style.Warning(text)
	}
	return style.OK(text)
}
