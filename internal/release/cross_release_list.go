package release

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/utils/colors"
	"github.com/olekukonko/tablewriter"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CrossReleaseList describes multiple releases across multiple environments
type CrossReleaseList struct {
	Environments []*environment.Environment
	Releases     map[string]*CrossRelease
}

// NewCrossReleaseList creates a new CrossReleaseList
func NewCrossReleaseList(environments []*environment.Environment) *CrossReleaseList {
	return &CrossReleaseList{
		Environments: environments,
		Releases:     make(map[string]*CrossRelease),
	}
}

// LoadCrossReleaseList loads all releases for given environments underneath the given base directory.
func LoadCrossReleaseList(baseDir string, environments []*environment.Environment, releaseFilter Filter) (*CrossReleaseList, error) {
	crossReleases := NewCrossReleaseList(environments)
	for _, env := range environments {
		// Load releases for given environment
		envDir := filepath.Join(baseDir, env.Name)
		envReleases, err := LoadAllInDir(envDir, releaseFilter)
		if err != nil {
			return nil, fmt.Errorf("loading releases in %s: %w", envDir, err)
		}

		// Add releases to cross-release list
		for _, rel := range envReleases {
			err := crossReleases.AddRelease(rel, env)
			if err != nil {
				return nil, fmt.Errorf("adding release %s: %w", rel.Name, err)
			}
		}
	}
	return crossReleases, nil
}

// GetEnvironmentIndex returns the index of the environment with the given name or -1 if not found.
func (r *CrossReleaseList) GetEnvironmentIndex(name string) int {
	for i, env := range r.Environments {
		if env.Name == name {
			return i
		}
	}
	return -1
}

// AddRelease adds a release for given environment.
func (r *CrossReleaseList) AddRelease(rel *Release, environment *environment.Environment) error {
	index := r.GetEnvironmentIndex(environment.Name)
	if index == -1 {
		return fmt.Errorf("environment %s not found in list", environment.Name)
	}

	crossRelease, ok := r.Releases[rel.Name]
	if !ok {
		crossRelease = NewCrossRelease(rel.Name, r.Environments)
		r.Releases[rel.Name] = crossRelease
	}
	crossRelease.Releases[index] = rel
	return nil
}

func (r *CrossReleaseList) SortedCrossReleases() []*CrossRelease {
	var releases []*CrossRelease
	for _, rel := range r.Releases {
		releases = append(releases, rel)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Name < releases[j].Name
	})
	return releases
}

// OnlySpecificReleases returns a subset of the releases in this list that match the given names.
func (r *CrossReleaseList) OnlySpecificReleases(releases []string) *CrossReleaseList {
	subset := NewCrossReleaseList(r.Environments)
	for _, rel := range releases {
		subset.Releases[rel] = r.Releases[rel]
	}
	return subset
}

func (r *CrossReleaseList) OnlyPromotableReleases() *CrossReleaseList {
	subset := NewCrossReleaseList(r.Environments)
	for _, rel := range r.Releases {
		if rel.Promotable() {
			subset.Releases[rel.Name] = rel
		}
	}
	return subset
}

type PrintOpts struct {
	// IsPromoting allows to dim releases that are not promotable (i.e. have no release in first environment)
	IsPromoting bool
}

// Print displays all releases versions across environments in a table format.
func (r *CrossReleaseList) Print(opts PrintOpts) {
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

	for _, crossRelease := range r.SortedCrossReleases() {
		// Check if releases and their values are synced across all environments
		releasesSynced := crossRelease.AllReleasesSynced()
		valuesSynced := crossRelease.AllValuesSynced()
		dimmed := opts.IsPromoting && !crossRelease.Promotable()

		row := []string{colorize(crossRelease.Name, releasesSynced, valuesSynced, dimmed)}
		for _, rel := range crossRelease.Releases {
			text := "-"
			if rel != nil && !rel.Missing {
				text = rel.Spec.Version
			}
			text = colorize(text, releasesSynced, valuesSynced, dimmed)
			row = append(row, text)
		}
		table.Append(row)
	}

	table.Render()
}

func colorize(text string, releasesSynced, valuesSynced, dimmed bool) string {
	if dimmed {
		return colors.InDarkGrey(text)
	}
	if !releasesSynced || !valuesSynced {
		if !releasesSynced {
			return color.InRed(text)
		} else {
			return color.InYellow(text)
		}
	}
	return color.InGreen(text)
}
