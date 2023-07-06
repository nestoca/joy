package promotion

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nestoca/joy-cli/internal/colors"
	"github.com/nestoca/joy-cli/internal/release"
	"strings"
)

var MajorSeparator = strings.Repeat("—", 80)
var MinorSeparator = ""

func preview(list *release.CrossReleaseList) error {
	releases := list.SortedCrossReleases()
	env := list.Environments[1]
	anyUnsynced := false

	for _, rel := range releases {
		// Skip releases that are not promotable because they are not in source environment
		if !rel.Promotable() {
			continue
		}

		// Check if releases and values are synced across all environments
		allReleasesSynced := rel.AllReleasesSynced()
		allValuesSynced := rel.AllValuesSynced()
		if allReleasesSynced && allValuesSynced {
			continue
		}
		anyUnsynced = true

		// Print header
		fmt.Println(MajorSeparator)
		fmt.Printf("🚀 %s %s/%s\n",
			color.InWhite("Release"),
			colors.InDarkYellow(env.Name),
			color.InBold(color.InYellow(rel.Name)))
		fmt.Println(MinorSeparator)
		source := rel.Releases[0]
		target := rel.Releases[1]

		// Determine operation
		operation := "Update"
		if target.Missing {
			operation = color.InBold("Create new")
		}
		operation = color.InYellow(operation)

		// Print release diff
		sections := 0
		if !allReleasesSynced {
			fmt.Printf("🕹  %s %s %s\n", operation, color.InWhite("release file"), colors.InDarkGrey(target.ReleaseFile.FilePath))
			err := printDiff(source.ReleaseFile, target.ReleaseFile, target.Missing)
			if err != nil {
				return fmt.Errorf("printing release diff: %w", err)
			}
			sections++
		}

		// Print values diff
		if !allValuesSynced {
			if sections > 0 {
				fmt.Println(MinorSeparator)
			}
			fmt.Printf("🎛  %s %s %s\n", operation, color.InWhite("values file"), colors.InDarkGrey(target.ValuesFile.FilePath))
			err := printDiff(source.ValuesFile, target.ValuesFile, target.Missing)
			if err != nil {
				return fmt.Errorf("printing values diff: %w", err)
			}
		}
	}

	if !anyUnsynced {
		fmt.Println("🎉 All releases are in sync!")
	}
	fmt.Println(MajorSeparator)
	return nil
}

func printDiff(source, target *release.YamlFile, targetMissing bool) error {
	merged := Merge(source.Tree, target.Tree)

	beforeYaml, err := target.ToYaml()
	if err != nil {
		return fmt.Errorf("marshalling before: %w", err)
	}

	afterYaml, err := release.TreeToYaml(merged, target.Indent)
	if err != nil {
		return fmt.Errorf("marshalling after: %w", err)
	}

	// If target is missing, we want to show the whole file as added
	if targetMissing {
		beforeYaml = ""
	}

	edits := myers.ComputeEdits(span.URIFromPath(""), beforeYaml, afterYaml)
	unified := fmt.Sprintf("%s", gotextdiff.ToUnified("before", "after", string(beforeYaml), edits))
	unified = strings.ReplaceAll(unified, "\\ No newline at end of file\n", "")
	unified = formatDiff(unified)

	fmt.Println(unified)
	return nil
}

func formatDiff(diff string) string {
	var coloredDiff strings.Builder
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "@@") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			coloredDiff.WriteString(color.InRed(line))
		} else if strings.HasPrefix(line, "+") {
			coloredDiff.WriteString(color.InGreen(line))
		} else {
			coloredDiff.WriteString(line)
		}
		coloredDiff.WriteString("\n")
	}
	return strings.TrimSpace(coloredDiff.String())
}
