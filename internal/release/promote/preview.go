package promote

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/utils/colors"
	"github.com/nestoca/joy/internal/yml"
	"strings"
)

var Separator = strings.Repeat("—", 80)

func preview(list *release.CrossReleaseList) error {
	releases := list.SortedCrossReleases()
	env := list.Environments[1]
	anyUnsynced := false

	for _, rel := range releases {
		// Skip releases that are not promotable because they are not in source environment
		if !rel.Promotable() {
			continue
		}

		// Skip releases that are synced across all environments
		allReleasesSynced := rel.AllReleasesSynced()
		if allReleasesSynced {
			continue
		}
		anyUnsynced = true
		source := rel.Releases[0]
		target := rel.Releases[1]

		// Determine operation
		operation := "Update release"
		if target.Missing {
			operation = "Create new release"
		}

		// Print release diff
		fmt.Println(Separator)
		fmt.Printf("🚀 %s %s/%s %s\n",
			operation,
			color.InGreen(env.Name),
			color.InBold(color.InYellow(target.Name)),
			colors.InDarkGrey("("+target.File.Path+")"))
		err := printDiff(source.File, target.File, target.Missing)
		if err != nil {
			return fmt.Errorf("printing release diff: %w", err)
		}
	}

	if !anyUnsynced {
		fmt.Println("🍺 All releases are in sync!")
	}
	fmt.Println(Separator)
	return nil
}

func printDiff(source, target *yml.File, targetMissing bool) error {
	merged := yml.Merge(source.Tree, target.Tree)

	beforeYaml, err := target.ToYaml()
	if err != nil {
		return fmt.Errorf("marshalling before: %w", err)
	}

	afterYaml, err := yml.TreeToYaml(merged, target.Indent)
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
