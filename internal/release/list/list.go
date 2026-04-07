package list

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/output"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type Params struct {
	Environments         []string
	ReferenceEnvironment string
}

const (
	UnknownStatus    = "unknown"
	PrereleaseStatus = "prerelease"
	BehindStatus     = "behind"
	AheadStatus      = "ahead"
	InSyncStatus     = "in-sync"
)

type Release struct {
	*v1alpha1.Release `json:",inline"`
	DisplayVersion    string `json:"version"`
	Status            string `json:"status"`
	Environment       string `json:"environment"`
}

type CrossRelease struct {
	Name     string    `json:"name"`
	Releases []Release `json:"releases"`
}

type ReleaseList struct {
	Environments         []string       `json:"environments"`
	ReferenceEnvironment string         `json:"referenceEnvironment"`
	CrossReleases        []CrossRelease `json:"crossReleases"`
}

func GetReleaseList(cat *catalog.Catalog, params Params) (ReleaseList, error) {
	releaseList := ReleaseList{
		ReferenceEnvironment: params.ReferenceEnvironment,
	}

	var selectedEnvIndices []int
	for i, env := range cat.Releases.Environments {
		if len(params.Environments) > 0 && !slices.Contains(params.Environments, env.Name) {
			continue
		}
		selectedEnvIndices = append(selectedEnvIndices, i)
		releaseList.Environments = append(releaseList.Environments, env.Name)
	}

	for _, crossRelease := range cat.Releases.Items {
		outputRelease := CrossRelease{
			Name: crossRelease.Name,
		}

		referenceVersion := NoReleaseVersion
		for _, rel := range crossRelease.Releases {
			if rel != nil && rel.Environment.Name == params.ReferenceEnvironment {
				referenceVersion = GetReleaseDisplayVersion(rel)
			}
		}

		for envIndex, rel := range crossRelease.Releases {
			if !slices.Contains(selectedEnvIndices, envIndex) {
				continue
			}
			displayVersion := GetReleaseDisplayVersion(rel)

			outputRelease.Releases = append(outputRelease.Releases, Release{
				Release:        rel,
				DisplayVersion: displayVersion,
				Status:         getVersionStatus(displayVersion, referenceVersion),
				Environment:    cat.Environments[envIndex].Name,
			})
		}

		releaseList.CrossReleases = append(releaseList.CrossReleases, outputRelease)
	}

	return releaseList, nil
}

// releasesByEnvironment groups releases by environment.
func releasesByEnvironment(releaseList ReleaseList) map[string][]*v1alpha1.Release {
	out := make(map[string][]*v1alpha1.Release)
	for _, cross := range releaseList.CrossReleases {
		for _, rel := range cross.Releases {
			if rel.Release != nil {
				out[rel.Environment] = append(out[rel.Environment], rel.Release)
			}
		}
	}
	return out
}

// flatReleases returns releases as a single slice (no grouping by environment).
// Used for JSON/YAML formats when there is exactly one environment.
func flatReleases(releaseList ReleaseList) []*v1alpha1.Release {
	out := []*v1alpha1.Release{}
	for _, cross := range releaseList.CrossReleases {
		for _, rel := range cross.Releases {
			if rel.Release != nil {
				out = append(out, rel.Release)
			}
		}
	}
	return out
}

func Render(writer io.Writer, releaseList ReleaseList, format output.Format, maxColumnWidth int) error {
	getReleases := func() any {
		if len(releaseList.Environments) == 1 {
			return flatReleases(releaseList)
		}
		return releasesByEnvironment(releaseList)
	}

	switch format {
	case output.FormatJson:
		return output.RenderJson(writer, getReleases())
	case output.FormatYaml:
		return output.RenderYaml(writer, getReleases())
	case output.FormatNames:
		return renderNames(writer, releaseList)
	case output.FormatTable:
		return renderTable(writer, releaseList, releaseList.ReferenceEnvironment, maxColumnWidth)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func renderNames(writer io.Writer, releaseList ReleaseList) error {
	uniqueNames := make(map[string]bool)
	for _, crossRelease := range releaseList.CrossReleases {
		for _, rel := range crossRelease.Releases {
			if rel.Release != nil {
				uniqueNames[rel.Release.Name] = true
				break
			}
		}
	}

	names := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		if _, err := fmt.Fprintln(writer, name); err != nil {
			return err
		}
	}
	return nil
}

func renderTable(writer io.Writer, releaseList ReleaseList, referenceEnvironment string, maxColumnWidth int) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	headers := table.Row{"NAME"}
	for _, env := range releaseList.Environments {
		headers = append(headers, strings.ToUpper(env))
	}
	t.AppendHeader(headers)

	legend := table.NewWriter()
	legend.SetStyle(table.StyleRounded)
	legend.AppendRow(table.Row{
		"Reference Environment: " + referenceEnvironment,
		style.DirtyVersion("Pre-Release (PR)"),
		style.BehindVersion("Behind"),
		style.AheadVersion("Ahead"),
		style.InSyncVersion("In-Sync"),
	})
	fmt.Println(legend.Render())

	for _, release := range releaseList.CrossReleases {
		row := table.Row{release.Name}
		for _, version := range release.Releases {
			displayVersion := version.DisplayVersion
			if maxColumnWidth != 0 && len(version.DisplayVersion) > maxColumnWidth {
				displayVersion = displayVersion[:maxColumnWidth-3] + "..."
			}
			displayVersion = colorizeVersion(displayVersion, version.Status)
			row = append(row, displayVersion)
		}
		t.AppendRow(row)
	}

	rendered := t.Render()
	_, err := io.WriteString(writer, rendered+"\n")
	if err != nil {
		return fmt.Errorf("writing release list as table: %w", err)
	}
	return nil
}

const (
	NoReleaseVersion = "-"
	NoVersion        = "no version"
)

func getVersionStatus(version, referenceVersion string) string {
	if version == NoReleaseVersion || version == NoVersion ||
		referenceVersion == NoReleaseVersion || referenceVersion == NoVersion {
		return UnknownStatus
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !strings.HasPrefix(referenceVersion, "v") {
		referenceVersion = "v" + referenceVersion
	}

	switch semver.Compare(version, referenceVersion) {
	case -1:
		if semver.Prerelease(version)+semver.Build(version) != "" {
			return PrereleaseStatus
		} else {
			return BehindStatus
		}
	case 1:
		return AheadStatus
	default:
		return InSyncStatus
	}
}

func colorizeVersion(version, status string) string {
	switch status {
	case PrereleaseStatus:
		return style.DirtyVersion(version)
	case BehindStatus:
		return style.BehindVersion(version)
	case AheadStatus:
		return style.AheadVersion(version)
	case InSyncStatus:
		return style.InSyncVersion(version)
	default:
		return version
	}
}

func GetReleaseDisplayVersion(rel *v1alpha1.Release) string {
	if rel == nil {
		return NoReleaseVersion
	}
	version := rel.Spec.Version
	if version == "" {
		return NoVersion
	}
	return version
}
