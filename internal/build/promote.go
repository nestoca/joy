package build

import (
	"fmt"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

type Opts struct {
	Catalog      *catalog.Catalog
	Writer       yml.Writer
	Environment  string
	Project      string
	Version      string
	ChartVersion string
}

func Promote(opts Opts) error {
	if !opts.Catalog.Environments[0].Spec.Promotion.FromPullRequests {
		version := "v" + opts.Version
		if semver.Prerelease(version)+semver.Build(version) != "" {
			return fmt.Errorf("cannot promote prerelease version to %s environment", opts.Environment)
		}
	}

	promotionCount := 0
	for _, crossRelease := range opts.Catalog.Releases.Items {
		release := crossRelease.Releases[0]
		if release == nil || release.Spec.Project != opts.Project {
			continue
		}

		versionNode, err := yml.FindNode(release.File.Tree, "spec.version")
		if err != nil {
			return fmt.Errorf("release %s has no version property: %w", release.Name, err)
		}

		versionNode.Value = opts.Version
		if err := release.File.UpdateYamlFromTree(); err != nil {
			return fmt.Errorf("updating release yaml from node tree: %w", err)
		}

		if opts.ChartVersion != "" {
			chartVersionNode, err := yml.FindNode(release.File.Tree, "spec.chart.version")
			if err != nil {
				return fmt.Errorf("release %s has no chart version property: %w", release.Name, err)
			}
			chartVersionNode.Value = opts.ChartVersion
			if err := release.File.UpdateYamlFromTree(); err != nil {
				return fmt.Errorf("updating release yaml from node tree: %w", err)
			}
		}

		if err := opts.Writer.WriteFile(release.File); err != nil {
			return fmt.Errorf("writing release file: %w", err)
		}

		if opts.ChartVersion != "" {
			fmt.Printf("✅ Promoted release %s to version %s and chart version %s\n", style.Resource(release.Name), style.Version(opts.Version), style.Version(opts.ChartVersion))
		} else {
			fmt.Printf("✅ Promoted release %s to version %s\n", style.Resource(release.Name), style.Version(opts.Version))
		}
		promotionCount++
	}

	if promotionCount == 0 {
		return fmt.Errorf("no releases found for project %s", opts.Project)
	}
	plural := ""
	if promotionCount > 1 {
		plural = "s"
	}

	if opts.ChartVersion != "" {
		fmt.Printf("🍺 Promoted %d release%s of project %s in environment %s to version %s and chart version %s\n", promotionCount, plural, style.Resource(opts.Project), style.Resource(opts.Environment), style.Version(opts.Version), style.Version(opts.ChartVersion))
	} else {
		fmt.Printf("🍺 Promoted %d release%s of project %s in environment %s to version %s\n", promotionCount, plural, style.Resource(opts.Project), style.Resource(opts.Environment), style.Version(opts.Version))
	}
	return nil
}
