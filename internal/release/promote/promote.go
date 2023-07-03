package promote

import (
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy-cli/internal/git"
	"github.com/nestoca/joy-cli/internal/release"
	"github.com/nestoca/joy-cli/internal/release/cross"
	"strings"
)

func Promote(list *cross.ReleaseList, push bool) error {
	if len(list.Environments) != 2 {
		return fmt.Errorf("expecting 2 environments, got %d", len(list.Environments))
	}

	releases := list.SortedReleases()
	var promotedFiles []string
	var messages []string
	promotedReleaseCount := 0
	sourceEnv := list.Environments[0]
	targetEnv := list.Environments[1]

	for _, rel := range releases {
		// Check if releases and values are synced across all environments
		allReleasesSynced := rel.AllReleasesSynced()
		allValuesSynced := rel.AllValuesSynced()
		if allReleasesSynced && allValuesSynced {
			continue
		}
		promotedReleaseCount++

		source := rel.Releases[0]
		target := rel.Releases[1]

		// Promote release
		if !allReleasesSynced {
			fmt.Printf("%s %s\n", color.InWhite("🕹Promoting release file"), color.Colorize(darkGrey, target.ReleaseFile.FilePath))
			err := promoteFile(source.ReleaseFile, target.ReleaseFile)
			if err != nil {
				return fmt.Errorf("promoting release file %q: %w", target.ReleaseFile.FilePath, err)
			}
			promotedFiles = append(promotedFiles, target.ReleaseFile.FilePath)
		}

		// Promote values
		if !allValuesSynced {
			fmt.Printf("%s %s\n", color.InWhite("🎛Promoting values file"), color.Colorize(darkGrey, target.ValuesFile.FilePath))
			err := promoteFile(source.ValuesFile, target.ValuesFile)
			if err != nil {
				return fmt.Errorf("promoting values file %q: %w", target.ValuesFile.FilePath, err)
			}
			promotedFiles = append(promotedFiles, target.ValuesFile.FilePath)
		}

		// Determine release-specific message
		message := getReleaseMessage(rel.Name, source.Spec.Version, target.Spec.Version, allReleasesSynced, allValuesSynced)
		messages = append(messages, message)
	}

	if len(promotedFiles) > 0 {
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
		fmt.Println("✅Committed with message:")
		fmt.Println(message)

		// Commit and push
		if push {
			err = git.Push()
			if err != nil {
				return fmt.Errorf("pushing changes: %w", err)
			}
			fmt.Println("✅Pushed")
		} else {
			fmt.Println("👉Skipping push! (use `joy push` to push changes manually)")
		}
		fmt.Println(MajorSeparator)
		fmt.Println("🎉Promotion complete!")
	} else {
		fmt.Println("🎉Nothing to do, all releases already in sync!")
		return nil
	}

	return nil
}

func getReleaseMessage(releaseName, sourceVersion, targetVersion string, allReleasesSynced, allValuesSynced bool) string {
	versionChanged := ""
	if sourceVersion != targetVersion {
		versionChanged = fmt.Sprintf(" %s -> %s", targetVersion, sourceVersion)
	}
	return fmt.Sprintf("Promote %s%s", releaseName, versionChanged)
}

func getCommitMessage(sourceEnv, targetEnv string, promotedReleaseCount int, messages []string) string {
	if len(messages) == 1 {
		// Put details of single promotion on first and only line
		return fmt.Sprintf("%s (%s -> %s)", messages[0], sourceEnv, targetEnv)
	}

	// Put details of individual promotions on subsequent lines
	return fmt.Sprintf("Promote %d releases (%s -> %s)\n%s", promotedReleaseCount, sourceEnv, targetEnv, strings.Join(messages, "\n"))
}

func promoteFile(source, target *release.YamlFile) error {
	mergedTree := Merge(source.Tree, target.Tree)
	merged, err := target.CopyWithNewTree(mergedTree)
	if err != nil {
		return fmt.Errorf("copying target file to merged file: %w", err)
	}
	err = merged.Write()
	if err != nil {
		return fmt.Errorf("writing merged file: %w", err)
	}
	return nil
}
