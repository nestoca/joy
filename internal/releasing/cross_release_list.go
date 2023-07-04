package releasing

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/environment"
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
	releases := NewCrossReleaseList(environments)
	for _, env := range environments {
		envDir := filepath.Join(baseDir, env.Name)
		envReleases, err := LoadAllInDir(envDir, releaseFilter)
		if err != nil {
			return nil, fmt.Errorf("loading releases in %s: %w", envDir, err)
		}
		for _, rel := range envReleases {
			err := releases.AddRelease(rel, env)
			if err != nil {
				return nil, fmt.Errorf("adding release %s: %w", rel.Name, err)
			}
		}
	}
	return releases, nil
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
func (r *CrossReleaseList) AddRelease(release *Release, environment *environment.Environment) error {
	index := r.GetEnvironmentIndex(environment.Name)
	if index == -1 {
		return fmt.Errorf("environment %s not found in list", environment.Name)
	}

	rel, ok := r.Releases[release.Name]
	if !ok {
		rel = NewCrossRelease(release.Name, r.Environments)
		r.Releases[release.Name] = rel
	}
	rel.Releases[index] = release
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

func (r *CrossReleaseList) FilteredSpecificReleases(releases []string) *CrossReleaseList {
	subset := NewCrossReleaseList(r.Environments)
	for _, rel := range releases {
		subset.Releases[rel] = r.Releases[rel]
	}
	return subset
}

func (r *CrossReleaseList) Print() {
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

	for _, release := range r.SortedCrossReleases() {
		// Check if releases and their values are synced across all environments
		releasesSynced := release.AllReleasesSynced()
		valuesSynced := release.AllValuesSynced()

		row := []string{colorize(release.Name, releasesSynced, valuesSynced)}
		for _, rel := range release.Releases {
			text := "-"
			if rel != nil {
				text = rel.Spec.Version
			}
			text = colorize(text, releasesSynced, valuesSynced)
			row = append(row, text)
		}
		table.Append(row)
	}

	table.Render()
}

func colorize(text string, releasesSynced, valuesSynced bool) string {
	if !releasesSynced || !valuesSynced {
		if !releasesSynced {
			return color.InRed(text)
		} else {
			return color.InYellow(text)
		}
	}
	return color.InGreen(text)
}
