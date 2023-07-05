package promotion

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/colors"
	"github.com/nestoca/joy-cli/internal/git"
	"github.com/nestoca/joy-cli/internal/releasing"
	"strings"
)

// promote performs the promotion of all releases in given list.
func promote(list *releasing.CrossReleaseList, push bool) error {
	if len(list.Environments) != 2 {
		return fmt.Errorf("expecting 2 environments, got %d", len(list.Environments))
	}

	crossReleases := list.SortedCrossReleases()
	var promotedFiles []string
	var messages []string
	promotedReleaseCount := 0
	sourceEnv := list.Environments[0]
	targetEnv := list.Environments[1]

	for _, crossRelease := range crossReleases {
		// Check if releases and values are synced across all environments
		allReleasesSynced := crossRelease.AllReleasesSynced()
		allValuesSynced := crossRelease.AllValuesSynced()
		if allReleasesSynced && allValuesSynced {
			continue
		}
		promotedReleaseCount++

		source := crossRelease.Releases[0]
		target := crossRelease.Releases[1]

		// Determine operation
		operation := "Updating"
		if target.Missing {
			operation = color.InBold("Creating new")
		}
		operation = color.InYellow(operation)

		// Promote release
		if !allReleasesSynced {
			fmt.Printf("🕹  %s %s %s\n", operation, color.InWhite("release file"), colors.InDarkGrey(target.ReleaseFile.FilePath))
			err := promoteFile(source.ReleaseFile, target.ReleaseFile)
			if err != nil {
				return fmt.Errorf("promoting release file %q: %w", target.ReleaseFile.FilePath, err)
			}
			promotedFiles = append(promotedFiles, target.ReleaseFile.FilePath)
		}

		// Promote values
		if !allValuesSynced {
			fmt.Printf("🎛  %s %s %s\n", operation, color.InWhite("values file"), colors.InDarkGrey(target.ValuesFile.FilePath))
			err := promoteFile(source.ValuesFile, target.ValuesFile)
			if err != nil {
				return fmt.Errorf("promoting values file %q: %w", target.ValuesFile.FilePath, err)
			}
			promotedFiles = append(promotedFiles, target.ValuesFile.FilePath)
		}

		// Determine release-specific message
		message := getPromotionMessage(crossRelease.Name, source.Spec.Version, target.Spec.Version, target.Missing)
		messages = append(messages, message)
	}

	// Any files promoted?
	if len(promotedFiles) > 0 {
		// Commit changes
		fmt.Println(MajorSeparator)
		err := git.Add(promotedFiles)
		if err != nil {
			return fmt.Errorf("adding files to index: %w", err)
		}
		message := getCommitMessage(sourceEnv.Name, targetEnv.Name, promotedReleaseCount, messages)
		err = git.Commit(message)
		if err != nil {
			return fmt.Errorf("committing changes: %w", err)
		}
		fmt.Println("✅ Committed with message:")
		fmt.Println(message)

		// Push changes
		if push {
			err = git.Push()
			if err != nil {
				return fmt.Errorf("pushing changes: %w", err)
			}
			fmt.Println("✅ Pushed")
		} else {
			fmt.Println("👉 Skipping push! (use `joy push` to push changes manually)")
		}
		fmt.Println(MajorSeparator)
		fmt.Println("🎉 Promotion complete!")
	} else {
		fmt.Println("🎉 Nothing to do, all releases already in sync!")
		return nil
	}

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

// getCommitMessage computes the commit message for the whole promotion operation including all releases
func getCommitMessage(sourceEnv, targetEnv string, promotedReleaseCount int, messages []string) string {
	if len(messages) == 1 {
		// Put details of single promotion on first and only line
		return fmt.Sprintf("%s (%s -> %s)", messages[0], sourceEnv, targetEnv)
	}

	// Put details of individual promotions on subsequent lines
	return fmt.Sprintf("Promote %d releases (%s -> %s)\n%s", promotedReleaseCount, sourceEnv, targetEnv, strings.Join(messages, "\n"))
}

// promoteFile merges a specific source yaml release or values file onto an equivalent target file
func promoteFile(source, target *releasing.YamlFile) error {
	mergedTree := Merge(source.Tree, target.Tree)
	merged, err := target.CopyWithNewTree(mergedTree)
	if err != nil {
		return fmt.Errorf("making in-memory copy of target file using merged result: %w", err)
	}
	err = merged.Write()
	if err != nil {
		return fmt.Errorf("writing merged file: %w", err)
	}
	return nil
}
