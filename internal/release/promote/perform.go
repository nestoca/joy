package promote

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"strings"
)

// perform performs the promotion of all releases in given list and returns PR url if any
func (p *Promotion) perform(list *cross.ReleaseList) (string, error) {
	if len(list.Environments) != 2 {
		return "", fmt.Errorf("expecting 2 environments, got %d", len(list.Environments))
	}

	crossReleases := list.SortedCrossReleases()
	var promotedFiles []string
	var messages []string
	var promotedReleaseNames []string
	sourceEnv := list.Environments[0]
	targetEnv := list.Environments[1]

	for _, crossRelease := range crossReleases {
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
		err := p.yamlWriter.Write(promotedFile)
		if err != nil {
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
	prTitle, prBody := getPRTitleAndBody(message)
	prURL, err := p.pullRequestProvider.Create(branchName, prTitle, prBody)
	if err != nil {
		return "", fmt.Errorf("creating pull request: %w", err)
	}
	p.promptProvider.PrintPullRequestCreated(prURL)

	err = p.gitProvider.CheckoutMasterBranch()
	if err != nil {
		return "", err
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
