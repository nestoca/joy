package promote

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nestoca/joy-cli/internal/release/cross"
	"gopkg.in/yaml.v3"
	"strings"
)

var MajorSeparator = strings.Repeat("—", 80)
var MinorSeparator = ""

func Preview(list *cross.ReleaseList) error {
	releases := list.SortedReleases()
	env := list.Environments[1]
	anyUnsynced := false

	for _, rel := range releases {
		// Check if releases and values are synced across all environments
		allReleasesSynced := rel.AllReleasesSynced()
		allValuesSynced := rel.AllValuesSynced()
		if allReleasesSynced && allValuesSynced {
			continue
		}
		anyUnsynced = true

		// Print header
		fmt.Println(MajorSeparator)
		fmt.Printf("🚀%s %s/%s\n",
			color.InWhite("Release"),
			color.Colorize(darkYellow, env.Name),
			color.InBold(color.InYellow(rel.Name)))
		fmt.Println(MinorSeparator)
		source := rel.Releases[0]
		target := rel.Releases[1]

		// Print release diff
		sections := 0
		if !allReleasesSynced {
			fmt.Printf("%s %s\n", color.InWhite("🕹Release file"), color.Colorize(darkGrey, target.ReleaseFile.FilePath))
			err := printDiff(source.ReleaseFile.Tree, target.ReleaseFile.Tree)
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
			fmt.Printf("%s %s\n", color.InWhite("🎛Values file"), color.Colorize(darkGrey, target.ValuesFile.FilePath))
			err := printDiff(source.ValuesFile.Tree, target.ValuesFile.Tree)
			if err != nil {
				return fmt.Errorf("printing values diff: %w", err)
			}
		}
	}

	if !anyUnsynced {
		fmt.Println("🎉All releases are in sync!")
	}
	fmt.Println(MajorSeparator)
	return nil
}

func printDiff(source, target *yaml.Node) error {
	merged := Merge(source, target)

	beforeYaml, err := yaml.Marshal(target)
	if err != nil {
		return fmt.Errorf("marshalling before: %w", err)
	}

	afterYaml, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("marshalling after: %w", err)
	}

	edits := myers.ComputeEdits(span.URIFromPath(""), string(beforeYaml), string(afterYaml))
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
