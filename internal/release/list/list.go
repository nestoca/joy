package list

import (
	"fmt"
	"github.com/nestoca/joy/internal/git"
	"io"
	"os"
	"slices"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type Opts struct {
	// CatalogDir is the path to the catalog directory.
	CatalogDir string

	// SelectedEnvs is the list of environments that were selected by user to work with.
	SelectedEnvs []string

	// Filter specifies releases to list.
	// Optional, defaults to listing all releases.
	Filter filtering.Filter

	ReferenceEnvironment string

	// SkipCatalogUpdate skips catalog update and dirty check. Very useful for
	// troubleshooting and testing templates.
	SkipCatalogUpdate bool
}

func List(opts Opts) error {
	if opts.SkipCatalogUpdate {
		fmt.Println("ℹ️ Skipping catalog update and dirty check.")
	} else {
		if err := git.EnsureCleanAndUpToDateWorkingCopy(opts.CatalogDir); err != nil {
			return err
		}
	}

	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:             opts.CatalogDir,
		ReleaseFilter:   opts.Filter,
		SortEnvsByOrder: true,
	})
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

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

		io.WriteString(os.Stdout, legend.Render()+"\n")
	}

	for _, crossRelease := range cat.Releases.Items {
		var masterVersion string
		row := table.Row{crossRelease.Name}

		for i, rel := range crossRelease.Releases {
			displayVersion := GetReleaseDisplayVersion(rel)
			if rel != nil && rel.Environment.Name == opts.ReferenceEnvironment {
				masterVersion = "v" + displayVersion
			}
			if slices.Contains(disallowEnvIndexes, i) {
				continue
			}
			row = append(row, displayVersion)
		}

		if useColors {
			for i, value := range row {
				// The first index corresponds to the name which we do not want to colorize
				if i == 0 {
					continue
				}

				version, _ := value.(string)
				if version == "-" {
					continue
				}
				if !strings.HasPrefix(version, "v") {
					version = "v" + version
				}

				switch semver.Compare(version, masterVersion) {
				case -1:
					if semver.Prerelease(version)+semver.Build(version) != "" {
						row[i] = style.DirtyVersion(value)
					} else {
						row[i] = style.BehindVersion(value)
					}
				case 0:
					row[i] = style.InSyncVersion(value)
				case 1:
					row[i] = style.AheadVersion(value)
				}
			}
		}

		t.AppendRow(row)
	}

	io.WriteString(os.Stdout, t.Render()+"\n")

	return nil
}

func GetReleaseDisplayVersion(rel *v1alpha1.Release) string {
	if rel == nil {
		return "-"
	}
	version := rel.Spec.Version
	if version == "" {
		version = "no version"
	}
	return version
}
