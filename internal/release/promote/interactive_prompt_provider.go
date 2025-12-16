package promote

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nestoca/survey/v2"
	"github.com/nestoca/survey/v2/core"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
)

type InteractivePromptProvider struct {
	anyReleaseDiffPrinted bool
	out                   io.Writer
}

func NewInteractivePromptProvider(out io.Writer) *InteractivePromptProvider {
	return &InteractivePromptProvider{
		out: out,
	}
}

var Separator = strings.Repeat("—", 80)

const (
	sourceEnvIndex = 0
	targetEnvIndex = 1
)

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

func (i *InteractivePromptProvider) SelectReleases(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
	i.printf("%s", style.Notice("\n* indicates changes beyond just the version number\n\n"))

	sourceEnv := list.Environments[sourceEnvIndex]
	targetEnv := list.Environments[targetEnvIndex]

	// Format releases for user selection.
	var choices []string
	for _, crossRel := range list.Items {
		var choice string
		if crossRel.VersionInSync && crossRel.ValuesInSync {
			choice = fmt.Sprintf("%s\t%s",
				style.InSyncRelease(crossRel.Name),
				inSyncDisplayReleaseVersion(crossRel, maxColumnWidth))
		} else {
			choice = fmt.Sprintf("%s\t%s\t>\t%s\t",
				style.OutOfSyncRelease(crossRel.Name),
				outOfSyncDisplayReleaseVersionBefore(crossRel, maxColumnWidth),
				outOfSyncDisplayReleaseVersionAfter(crossRel, maxColumnWidth))
		}
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
		return cross.ReleaseList{}, fmt.Errorf("prompting for releases to promote: %w", err)
	}

	// Create new cross-release list with only the selected releases.
	var selectedReleaseNames []string
	for _, index := range selectedIndices {
		selectedReleaseNames = append(selectedReleaseNames, list.Items[index].Name)
	}
	return list.OnlySpecificReleases(selectedReleaseNames)
}

func inSyncDisplayReleaseVersion(crossRel *cross.Release, maxColumnWidth int) string {
	version := releaseDisplayVersion(crossRel.Releases[targetEnvIndex], true, maxColumnWidth)
	return style.InSyncReleaseVersion(version)
}

func outOfSyncDisplayReleaseVersionBefore(crossRel *cross.Release, maxColumnWidth int) string {
	version := releaseDisplayVersion(crossRel.Releases[targetEnvIndex], crossRel.ValuesInSync, maxColumnWidth)
	return style.DiffBefore(version)
}

func outOfSyncDisplayReleaseVersionAfter(crossRel *cross.Release, maxColumnWidth int) string {
	version := releaseDisplayVersion(crossRel.Releases[sourceEnvIndex], crossRel.ValuesInSync, maxColumnWidth)
	return style.DiffAfter(version)
}

func releaseDisplayVersion(rel *v1alpha1.Release, valuesInSync bool, maxColumnWidth int) string {
	version := "-"
	if rel != nil {
		version = rel.Spec.Version
	}
	if version == "" {
		version = "unversioned"
	}
	if !valuesInSync && version != "-" {
		version += "*"
	}
	if len(version) > maxColumnWidth {
		return version[:maxColumnWidth-3] + "..."
	}
	return version
}

// alignColumns formats the given lines based on tab separators and aligns the columns.
func alignColumns(lines []string) []string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, line)
	}
	_ = w.Flush()

	result := buf.String()
	result = strings.TrimSpace(result)

	return strings.Split(result, "\n")
}

func (i *InteractivePromptProvider) ConfirmCreatingPromotionPullRequest(autoMerge, draft bool) (bool, error) {
	var message string
	var ok bool

	if draft {
		message = "Creating a draft promotion pull request. Do you wish to continue?"
	} else if autoMerge {
		message = "Creating and auto-merging a promotion pull request. Do you wish to continue?"
	}

	err := survey.AskOne(&survey.Confirm{Message: message}, &ok)
	if err != nil {
		return false, fmt.Errorf("asking user for confirmation: %w", err)
	}

	return ok, nil
}

const (
	CreatePR    = "Create PR"
	CreateDraft = "Create Draft PR"
	ViewGitLog  = "View Git Log"
	Cancel      = "Cancel"
)

func (i *InteractivePromptProvider) SelectPromotionAction() (string, error) {
	var selectedAction string

	actions := []string{CreatePR, CreateDraft, ViewGitLog, Cancel}
	prompt := &survey.Select{
		Message: "What would you like to do?",
		Options: actions,
	}

	err := survey.AskOne(prompt, &selectedAction)
	if err != nil {
		return "", fmt.Errorf("asking user for PR state: %w", err)
	}

	return selectedAction, nil
}

func (*InteractivePromptProvider) ConfirmAutoMergePullRequest() (answer bool, err error) {
	err = survey.AskOne(&survey.Confirm{Message: "Do you want to auto-merge the resulting PR?"}, &answer)
	return
}

func (i *InteractivePromptProvider) PrintNoPromotableReleasesFound(releasesFiltered bool, sourceEnv *v1alpha1.Environment, targetEnv *v1alpha1.Environment) {
	matchingSelection := ""
	if releasesFiltered {
		matchingSelection = "in current selection "
	}
	i.printf("🤷 No releases found %sfor promoting from %s to %s.\n", matchingSelection, style.Resource(sourceEnv.Name), style.Resource(targetEnv.Name))
}

func (i *InteractivePromptProvider) PrintNoPromotableEnvironmentFound(environmentsFiltered bool) {
	matchingSelection := ""
	if environmentsFiltered {
		matchingSelection = "in current selection "
	}
	i.printf("🤷 No environments found %sfor promoting.\n", matchingSelection)
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
	i.println(Separator)
	i.printf("🚀 %s %s/%s %s\n",
		operation,
		style.ResourceEnvPrefix(targetEnvName),
		style.Resource(releaseName),
		style.SecondaryInfo("("+promotedFile.Path+")"))
	err := i.printDiff(existingTargetFile, promotedFile)
	if err != nil {
		return fmt.Errorf("printing release diff: %w", err)
	}
	return nil
}

func (i *InteractivePromptProvider) printDiff(before, after *yml.File) error {
	beforeYaml := ""
	if before != nil {
		beforeYaml = string(before.Yaml)
	}
	edits := myers.ComputeEdits(span.URIFromPath(""), beforeYaml, string(after.Yaml))
	unified := fmt.Sprintf("%s", gotextdiff.ToUnified("before", "after", beforeYaml, edits))
	unified = strings.ReplaceAll(unified, "\\ No newline at end of file\n", "")
	unified = formatDiff(unified)

	i.println(unified)
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
		i.println("🍺 All releases are in sync!")
	}
	i.println(Separator)
}

func (i *InteractivePromptProvider) PrintUpdatingTargetRelease(targetEnvName, releaseName, releaseFilePath string, isCreating bool) {
	operation := "Updating release"
	if isCreating {
		operation = "Creating new release"
	}

	i.printf("🚀 %s %s/%s %s\n",
		operation,
		style.ResourceEnvPrefix(targetEnvName),
		style.Resource(releaseName),
		style.SecondaryInfo("("+releaseFilePath+")"))
}

func (i *InteractivePromptProvider) PrintBranchCreated(branchName, message string) {
	i.printf("✅ Committed and pushed new branch %s with message:\n%s\n",
		style.Resource(branchName),
		style.SecondaryInfo(message))
}

func (i *InteractivePromptProvider) PrintDraftPullRequestCreated(url string) {
	i.printf("✅ Created draft pull request: %s\n", style.Link(url))
}

func (i *InteractivePromptProvider) PrintPullRequestCreated(url string) {
	i.printf("✅ Created pull request: %s\n", style.Link(url))
}

func (i *InteractivePromptProvider) PrintCanceled() {
	i.println("🛑 Operation cancelled, no harm done! 😅")
}

func (i *InteractivePromptProvider) PrintSelectedReleasesAlreadyInSync() {
	i.println("🍺 Nothing to do, selected releases already in sync!")
}

func (i *InteractivePromptProvider) PrintCompleted() {
	i.println("🍺 Promotion complete!")
}

func (i *InteractivePromptProvider) PrintSelectedNonPromotableReleases(invalidReleases, targetEnvName string) {
	i.printf("🚫 Cannot promote release(s): %s. Target environment %s does not allow non-standard versions.\n",
		style.Resource(invalidReleases),
		style.Resource(targetEnvName))
}

func (i *InteractivePromptProvider) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(i.out, format, args...)
}

func (i *InteractivePromptProvider) println(a ...any) {
	_, _ = fmt.Fprintln(i.out, a...)
}
