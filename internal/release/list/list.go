package list

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type Opts struct {
	// SelectedEnvs is the list of environments that were selected by user to work with.
	SelectedEnvs []string

	ReferenceEnvironment string

	MaxColumnWidth int
}

func List(cat *catalog.Catalog, opts Opts) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	var disallowEnvIndexes []int
	headers := table.Row{"NAME"}
	for i, env := range cat.Releases.Environments {
		if len(opts.SelectedEnvs) > 0 && !slices.Contains(opts.SelectedEnvs, env.Name) {
			disallowEnvIndexes = append(disallowEnvIndexes, i)
			continue
		}
		headers = append(headers, strings.ToUpper(env.Name))
	}

	t.AppendHeader(headers)

	var useColors bool
	for _, env := range cat.Environments {
		if env.Name == opts.ReferenceEnvironment {
			useColors = true
			break
		}
	}

	if useColors {
		legend := table.NewWriter()
		legend.SetStyle(table.StyleRounded)

		legend.AppendRow(table.Row{
			"Legend: reference-environment: " + opts.ReferenceEnvironment,
			style.DirtyVersion("pre-release (PR)"),
			style.BehindVersion("behind"),
			style.AheadVersion("ahead"),
			style.InSyncVersion("in-sync"),
		})

		fmt.Println(legend.Render())
	}

	for _, crossRelease := range cat.Releases.Items {
		row := table.Row{crossRelease.Name}

		var referenceVersion string
		for _, rel := range crossRelease.Releases {
			if rel != nil && rel.Environment.Name == opts.ReferenceEnvironment {
				referenceVersion = GetReleaseDisplayVersion(rel)
			}
		}

		for i, rel := range crossRelease.Releases {
			if slices.Contains(disallowEnvIndexes, i) {
				continue
			}
			displayVersion := GetReleaseDisplayVersion(rel)
			version := displayVersion
			if opts.MaxColumnWidth != 0 && len(displayVersion) > opts.MaxColumnWidth {
				displayVersion = displayVersion[:opts.MaxColumnWidth-3] + "..."
			}
			if useColors {
				displayVersion = colorizeVersion(displayVersion, version, referenceVersion)
			}
			row = append(row, displayVersion)
		}

		t.AppendRow(row)
	}

	fmt.Println(t.Render())

	return nil
}

const (
	NoReleaseVersion = "-"
	NoVersion        = "no version"
)

func colorizeVersion(text, version, referenceVersion string) string {
	if version == NoReleaseVersion || version == NoVersion ||
		referenceVersion == NoReleaseVersion || referenceVersion == NoVersion {
		return text
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
			return style.DirtyVersion(text)
		} else {
			return style.BehindVersion(text)
		}
	case 1:
		return style.AheadVersion(text)
	default:
		return style.InSyncVersion(text)
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
