package promote

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/cross"
)

type PerformParams struct {
	list      *cross.ReleaseList
	draft     bool
	autoMerge bool
}

// perform performs the promotion of all releases in given list and returns PR url if any
func (p *Promotion) perform(params PerformParams) (string, error) {
	if len(params.list.Environments) != 2 {
		return "", fmt.Errorf("expecting 2 environments, got %d", len(params.list.Environments))
	}

	var (
		promotedFiles        []string
		messages             []string
		promotedReleaseNames []string
	)

	sourceEnv := params.list.Environments[0]
	targetEnv := params.list.Environments[1]

	for _, crossRelease := range params.list.SortedCrossReleases() {
		// Skip releases already in sync
		promotedFile := crossRelease.PromotedFile
		if promotedFile == nil {
			continue
		}
		promotedReleaseNames = append(promotedReleaseNames, crossRelease.Name)

		// Update target release file
		sourceRelease := crossRelease.Releases[0]
		targetRelease := crossRelease.Releases[1]
		isCreatingTargetRelease := targetRelease == nil

		p.promptProvider.PrintUpdatingTargetRelease(targetEnv.Name, crossRelease.Name, promotedFile.Path, isCreatingTargetRelease)

		if err := p.yamlWriter.Write(promotedFile); err != nil {
			return "", fmt.Errorf("writing release %q promoted target yaml to file %q: %w", crossRelease.Name, promotedFile.Path, err)
		}

		promotedFiles = append(promotedFiles, promotedFile.Path)

		// Determine release-specific message
		message := getPromotionMessage(crossRelease.Name, sourceRelease, targetRelease)
		messages = append(messages, message)
	}

	// Nothing promoted?
	if len(promotedFiles) == 0 {
		return "", fmt.Errorf("no releases promoted, should not reach this point")
	}

	// Create new branch and commit and push changes
	branchName := getBranchName(sourceEnv.Name, targetEnv.Name, promotedReleaseNames)
	message := getCommitMessage(sourceEnv.Name, targetEnv.Name, promotedReleaseNames, messages)

	err := p.gitProvider.CreateAndPushBranchWithFiles(branchName, promotedFiles, message)
	if err != nil {
		return "", err
	}

	p.promptProvider.PrintBranchCreated(branchName, message)

	// Create pull request
	var labels []string
	if params.autoMerge {
		labels = append(labels, "auto-merge")
	}

	prTitle, prBody := getPRTitleAndBody(message)

	prURL, err := p.pullRequestProvider.Create(pr.CreateParams{
		Branch: branchName,
		Title:  prTitle,
		Body:   prBody,
		Labels: labels,
		Draft:  params.draft,
	})
	if err != nil {
		return "", fmt.Errorf("creating pull request: %w", err)
	}

	if params.draft {
		p.promptProvider.PrintDraftPullRequestCreated(prURL)
	}
	p.promptProvider.PrintPullRequestCreated(prURL)

	if err := p.gitProvider.CheckoutMasterBranch(); err != nil {
		return "", fmt.Errorf("checking out master: %w", err)
	}

	p.promptProvider.PrintCompleted()

	return prURL, nil
}

// getPromotionMessage computes the message for a specific release promotion
func getPromotionMessage(releaseName string, sourceRelease, targetRelease *v1alpha1.Release) string {
	previousVersion := "(missing)"
	if targetRelease != nil {
		previousVersion = targetRelease.Spec.Version
	}
	return fmt.Sprintf("Promote %s %s -> %s", releaseName, previousVersion, sourceRelease.Spec.Version)
}

func getBranchName(sourceEnv, targetEnv string, promotedReleaseNames []string) string {
	var releases string
	if len(promotedReleaseNames) == 1 {
		releases = promotedReleaseNames[0]
	} else {
		releases = fmt.Sprintf("%d-releases", len(promotedReleaseNames))
	}
	uniqueID := uuid.New().String()
	name := fmt.Sprintf("promote-%s-from-%s-to-%s-%s", releases, sourceEnv, targetEnv, uniqueID)
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

// getCommitMessage computes the commit message for the whole promotion operation including all releases
func getCommitMessage(sourceEnv, targetEnv string, promotedReleaseNames []string, messages []string) string {
	if len(messages) == 1 {
		// Put details of single promotion on first and only line
		return fmt.Sprintf("%s (%s -> %s)", messages[0], sourceEnv, targetEnv)
	}

	// Put details of individual promotions on subsequent lines
	return fmt.Sprintf("Promote %d releases (%s -> %s)\n%s", len(promotedReleaseNames), sourceEnv, targetEnv, strings.Join(messages, "\n"))
}

// getPRTitleAndBody computes the title and body for the pull request based on the commit message
func getPRTitleAndBody(commitMessage string) (string, string) {
	lines := strings.Split(commitMessage, "\n")
	title := lines[0]
	body := strings.Join(lines[1:], "\n")
	return title, body
}
