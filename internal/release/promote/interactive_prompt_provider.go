package promote

import (
	"bytes"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"strings"
	"text/tabwriter"
)

type InteractivePromptProvider struct {
	anyReleaseDiffPrinted bool
}

var Separator = strings.Repeat("—", 80)

func (i *InteractivePromptProvider) SelectSourceEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error) {
	var index int
	err := survey.AskOne(&survey.Select{
		Message: "Select source promotion environment",
		Options: v1alpha1.GetEnvironmentNames(environments),
	},
		&index,
		survey.WithPageSize(10),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for source environment: %w", err)
	}
	return environments[index], nil
}

func (i *InteractivePromptProvider) SelectTargetEnvironment(environments []*v1alpha1.Environment) (*v1alpha1.Environment, error) {
	var index int
	err := survey.AskOne(&survey.Select{
		Message: "Select target promotion environment",
		Options: v1alpha1.GetEnvironmentNames(environments),
	},
		&index,
		survey.WithPageSize(10),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for target environment: %w", err)
	}
	return environments[index], nil
}

func (i *InteractivePromptProvider) SelectReleases(list *cross.ReleaseList) (*cross.ReleaseList, error) {
	sourceEnv := list.Environments[0]
	targetEnv := list.Environments[1]

	// Format releases for user selection.
	var choices []string
	for _, item := range list.Items {
		inSync := item.AreVersionsInSync()
		choice := fmt.Sprintf("%s\t%s\t>\t%s\t",
			style.Resource(item.Name),
			cross.GetReleaseDisplayVersion(item.Releases[1], inSync),
			cross.GetReleaseDisplayVersion(item.Releases[0], inSync))
		choices = append(choices, choice)
	}
	choices = alignColumns(choices)

	// Transform allows to show only release name identifiers after user confirms selection,
	// instead of the full colorized, tabbed release name and versions string.
	transform := func(ans interface{}) interface{} {
		answers := ans.([]core.OptionAnswer)
		for i := range answers {
			answers[i].Value = list.Items[answers[i].Index].Name
		}
		return answers
	}

	// Prompt user to select releases to promote.
	prompt := &survey.MultiSelect{
		Message: fmt.Sprintf("Select releases to promote from %s to %s",
			style.Resource(sourceEnv.Name),
			style.Resource(targetEnv.Name)),
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
		selectedReleaseNames = append(selectedReleaseNames, list.Items[index].Name)
	}
	return list.OnlySpecificReleases(selectedReleaseNames), nil
}

// alignColumns formats the given lines based on tab separators and aligns the columns.
func alignColumns(lines []string) []string {
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
		result[i] = line
	}
	return result
}

func (i *InteractivePromptProvider) ConfirmCreatingPromotionPullRequest() (bool, error) {
	actions := []string{"Create pull request", "Cancel"}
	prompt := &survey.Select{
		Message: "Do you want to create a promotion pull request?",
		Options: actions,
	}
	var selectedAction string
	err := survey.AskOne(prompt, &selectedAction)
	if err != nil {
		return false, fmt.Errorf("asking user for confirmation: %w", err)
	}
	return selectedAction == actions[0], nil
}

func (i *InteractivePromptProvider) PrintNoPromotableReleasesFound(releasesFiltered bool, sourceEnv *v1alpha1.Environment, targetEnv *v1alpha1.Environment) {
	matchingSelection := ""
	if releasesFiltered {
		matchingSelection = "(matching your selection) "
	}
	fmt.Printf("🤷 No releases found %sfor promotion from %s to %s.\n", matchingSelection, style.Resource(sourceEnv), style.Resource(targetEnv))
}

func (i *InteractivePromptProvider) PrintNoPromotableEnvironmentFound(environmentsFiltered bool) {
	matchingSelection := ""
	if environmentsFiltered {
		matchingSelection = "(matching your selection) "
	}
	fmt.Printf("🤷 No environments found %sfor promotion.\n", matchingSelection)
}

func (i *InteractivePromptProvider) PrintStartPreview() {
	i.anyReleaseDiffPrinted = false
}

func (i *InteractivePromptProvider) PrintReleasePreview(targetEnvName string, releaseName string, existingTargetFile, promotedFile *yml.File) error {
	i.anyReleaseDiffPrinted = true

	// Determine operation
	operation := "Update release"
	if existingTargetFile == nil {
		operation = "Create new release"
	}

	// Print release diff
	fmt.Println(Separator)
	fmt.Printf("🚀 %s %s/%s %s\n",
		operation,
		style.ResourceEnvPrefix(targetEnvName),
		style.Resource(releaseName),
		style.SecondaryInfo("("+promotedFile.Path+")"))
	err := printDiff(existingTargetFile, promotedFile)
	if err != nil {
		return fmt.Errorf("printing release diff: %w", err)
	}
	return nil
}

func printDiff(before, after *yml.File) error {
	beforeYaml := ""
	if before != nil {
		beforeYaml = string(before.Yaml)
	}
	edits := myers.ComputeEdits(span.URIFromPath(""), beforeYaml, string(after.Yaml))
	unified := fmt.Sprintf("%s", gotextdiff.ToUnified("before", "after", beforeYaml, edits))
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
			coloredDiff.WriteString(style.DiffBefore(line))
		} else if strings.HasPrefix(line, "+") {
			coloredDiff.WriteString(style.DiffAfter(line))
		} else {
			coloredDiff.WriteString(line)
		}
		coloredDiff.WriteString("\n")
	}
	return strings.TrimSpace(coloredDiff.String())
}

func (i *InteractivePromptProvider) PrintEndPreview() {
	if !i.anyReleaseDiffPrinted {
		fmt.Println("🍺 All releases are in sync!")
	}
	fmt.Println(Separator)
}

func (i *InteractivePromptProvider) PrintUpdatingTargetRelease(targetEnvName, releaseName, releaseFilePath string, isCreating bool) {
	operation := "Updating release"
	if isCreating {
		operation = "Creating new release"
	}

	fmt.Printf("🚀 %s %s/%s %s\n",
		operation,
		style.ResourceEnvPrefix(targetEnvName),
		style.Resource(releaseName),
		style.SecondaryInfo("("+releaseFilePath+")"))
}

func (i *InteractivePromptProvider) PrintBranchCreated(branchName, message string) {
	fmt.Printf(
		"✅ Committed and pushed new branch %s with message:\n%s\n",
		style.Resource(branchName),
		style.SecondaryInfo(message))
}

func (i *InteractivePromptProvider) PrintPullRequestCreated(url string) {
	fmt.Printf("✅ Created pull request: %s\n", style.Hyperlink(url))
}

func (i *InteractivePromptProvider) PrintCanceled() {
	fmt.Println("🛑 Operation cancelled, no harm done! 😅")
}

func (i *InteractivePromptProvider) PrintSelectedReleasesAlreadyInSync() {
	fmt.Println("🍺 Nothing to do, selected releases already in sync!")
}

func (i *InteractivePromptProvider) PrintCompleted() {
	fmt.Println("🍺 Promotion complete!")
}
