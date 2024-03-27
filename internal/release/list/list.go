package list

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type Params struct {
	SelectedEnvs         []string
	ReferenceEnvironment string
}

const (
	UnknownStatus    = "unknown"
	PreReleaseStatus = "pre-release"
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
		if len(params.SelectedEnvs) > 0 && !slices.Contains(params.SelectedEnvs, env.Name) {
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

func FormatReleaseListAsTable(releaseList ReleaseList, referenceEnvironment string, maxColumnWidth int) string {
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

	return t.Render()
}

func FormatReleaseListAsJson(releaseList ReleaseList) (string, error) {
	b, err := json.MarshalIndent(releaseList, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling output as JSON: %w", err)
	}
	return string(b), nil
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
			return PreReleaseStatus
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
	case PreReleaseStatus:
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
