package promote

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/google/uuid"
	"github.com/nestoca/joy/internal/gh"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/utils/colors"
	"github.com/nestoca/joy/internal/yml"
	"strings"
)

// perform performs the promotion of all releases in given list.
func perform(list *release.CrossReleaseList) error {
	if len(list.Environments) != 2 {
		return fmt.Errorf("expecting 2 environments, got %d", len(list.Environments))
	}

	crossReleases := list.SortedCrossReleases()
	var promotedFiles []string
	var messages []string
	var promotedReleaseNames []string
	sourceEnv := list.Environments[0]
	targetEnv := list.Environments[1]

	for _, crossRelease := range crossReleases {
		// Check if releases and values are synced across all environments
		allReleasesSynced := crossRelease.AllReleasesSynced()
		if allReleasesSynced {
			continue
		}
		promotedReleaseNames = append(promotedReleaseNames, crossRelease.Name)

		source := crossRelease.Releases[0]
		target := crossRelease.Releases[1]

		// Determine operation
		operation := "Updating release"
		if target.Missing {
			operation = "Creating new release"
		}

		// Promote release
		fmt.Printf("🚀 %s %s/%s %s\n",
			operation,
			color.InGreen(targetEnv.Name),
			color.InBold(color.InYellow(target.Name)),
			colors.InDarkGrey("("+target.File.Path+")"))
		err := promoteFile(source.File, target.File)
		if err != nil {
			return fmt.Errorf("promoting release %q: %w", target.File.Path, err)
		}
		promotedFiles = append(promotedFiles, target.File.Path)

		// Determine release-specific message
		message := getPromotionMessage(crossRelease.Name, source.Spec.Version, target.Spec.Version, target.Missing)
		messages = append(messages, message)
	}

	// Nothing promoted?
	if len(promotedFiles) == 0 {
		fmt.Println("🍺 Nothing to do, all releases already in sync!")
		return nil
	}

	// Create branch
	branchName := getBranchName(sourceEnv.Name, targetEnv.Name, promotedReleaseNames)
	err := git.CreateBranch(branchName)
	if err != nil {
		return fmt.Errorf("creating branch %s: %w", branchName, err)
	}
	fmt.Printf("✅ Created branch: %s\n", branchName)

	// Commit changes
	err = git.Add(promotedFiles)
	if err != nil {
		return fmt.Errorf("adding files to index: %w", err)
	}
	message := getCommitMessage(sourceEnv.Name, targetEnv.Name, promotedReleaseNames, messages)
	err = git.Commit(message)
	if err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}
	fmt.Println("✅ Committed with message:")
	fmt.Println(message)

	// Push changes
	err = git.PushNewBranch(branchName)
	if err != nil {
		return fmt.Errorf("pushing changes: %w", err)
	}
	fmt.Println("✅ Pushed")

	// Create pull request
	prTitle, prBody := getPRTitleAndBody(message)
	err = gh.CreatePullRequest("pr", "create", "--title", prTitle, "--body", prBody)
	if err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}
	fmt.Printf("✅ Created pull request: %s\n", prTitle)

	// Checking out master branch
	err = git.Checkout("master")
	if err != nil {
		return fmt.Errorf("checking out master branch: %w", err)
	}
	fmt.Println("✅ Checked out master branch")

	fmt.Println("🍺 Promotion complete!")
	return nil
}

// getPromotionMessage computes the message for a specific release promotion
func getPromotionMessage(releaseName, sourceVersion, targetVersion string, missing bool) string {
	versionChanged := ""
	if missing {
		versionChanged = fmt.Sprintf(" (missing) -> %s", targetVersion)
	} else if sourceVersion != targetVersion {
		versionChanged = fmt.Sprintf(" %s -> %s", targetVersion, sourceVersion)
	}
	return fmt.Sprintf("Promote %s%s", releaseName, versionChanged)
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

// promoteFile merges a specific source yaml release or values file onto an equivalent target file
func promoteFile(source, target *yml.File) error {
	mergedTree := yml.Merge(source.Tree, target.Tree)
	merged, err := target.CopyWithNewTree(mergedTree)
	if err != nil {
		return fmt.Errorf("making in-memory copy of target file using merged result: %w", err)
	}
	err = merged.WriteYaml()
	if err != nil {
		return fmt.Errorf("writing merged file: %w", err)
	}
	return nil
}
