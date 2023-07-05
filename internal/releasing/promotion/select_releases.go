package promotion

import (
	"bytes"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/colors"
	"github.com/nestoca/joy-cli/internal/releasing"
	"strings"
	"text/tabwriter"
)

func SelectReleases(sourceEnv, targetEnv string, list *releasing.CrossReleaseList) (*releasing.CrossReleaseList, error) {
	// Format releases for user selection.
	var choices []string
	sortedCrossReleases := list.SortedCrossReleases()
	env := list.Environments[1]
	for _, rel := range sortedCrossReleases {
		rel := fmt.Sprintf("%s/%s\t%s\t>\t%s",
			colors.InDarkYellow(env.Name),
			color.InBold(color.InYellow(rel.Name)),
			color.InRed(GetReleaseVersion(rel.Releases[1])),
			color.InGreen(GetReleaseVersion(rel.Releases[0])))
		choices = append(choices, rel)
	}
	choices = AlignColumns(choices)

	// Transform allows to show only release name identifiers after user confirms selection,
	// instead of the full colorized, tabbed release name and versions string.
	transform := func(ans interface{}) interface{} {
		answers := ans.([]core.OptionAnswer)
		for i := range answers {
			answers[i].Value = sortedCrossReleases[answers[i].Index].Name
		}
		return answers
	}

	// Prompt user to select releases to promote.
	prompt := &survey.MultiSelect{
		Message: fmt.Sprintf("Select releases to promote from %s %s %s",
			color.InBold(color.InWhite(sourceEnv)),
			color.InBold("to"),
			color.InBold(color.InWhite(targetEnv))),
		Options: choices,
	}
	questions := []*survey.Question{{
		Prompt:    prompt,
		Transform: transform,
	}}
	var selectedIndices []int
	err := survey.Ask(questions,
		&selectedIndices,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required))
	if err != nil {
		return nil, fmt.Errorf("prompting for releases to promote: %w", err)
	}

	// Create new cross-release list with only the selected releases.
	var selectedReleaseNames []string
	for _, index := range selectedIndices {
		selectedReleaseNames = append(selectedReleaseNames, sortedCrossReleases[index].Name)
	}
	return list.OnlySpecificReleases(selectedReleaseNames), nil
}

func GetReleaseVersion(rel *releasing.Release) string {
	if rel == nil || rel.Missing {
		return "-"
	}
	return rel.Spec.Version
}

// AlignColumns formats the given lines based on tab separators and aligns the columns.
func AlignColumns(lines []string) []string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, line)
	}
	_ = w.Flush()
	formattedOutput := buf.String()

	// Convert the bytes to strings
	formattedLines := strings.Split(strings.TrimSpace(formattedOutput), "\n")
	result := make([]string, len(formattedLines))
	for i, line := range formattedLines {
		result[i] = string(line)
	}
	return result
}
