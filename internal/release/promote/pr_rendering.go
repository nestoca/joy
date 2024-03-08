package promote

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"
)

type ChangeType string

const (
	ChangeTypeUpgrade   ChangeType = "Upgrade"
	ChangeTypeDowngrade ChangeType = "Downgrade"
	ChangeTypeUpdate    ChangeType = "Update"
)

type ReleaseInfo struct {
	Name         string
	Project      *v1alpha1.Project
	CodeOwners   []string
	Repository   string
	Source       ReleaseWithGitTag
	Target       ReleaseWithGitTag
	OlderGitTag  string
	NewerGitTag  string
	IsPrerelease bool
	// IsUpgrade is now deprecated and kept only temporarily while we transition to ChangeType
	IsUpgrade  bool
	ChangeType ChangeType
	Commits    []*CommitInfo
	Error      error
}

type PromotionInfo struct {
	SourceEnvironment *v1alpha1.Environment
	TargetEnvironment *v1alpha1.Environment
	Releases          []*ReleaseInfo
	Error             error
}

type CommitInfo struct {
	Sha          string
	ShortSha     string
	Author       string
	GitHubAuthor string
	Message      string
	ShortMessage string
}

func getReleaseInfo(sourceRelease, targetRelease *v1alpha1.Release, opts PerformOpts) (*ReleaseInfo, error) {
	project := sourceRelease.Project
	changeType := ChangeTypeUpgrade
	if targetRelease != nil {
		comparison := semver.Compare("v"+sourceRelease.Spec.Version, "v"+targetRelease.Spec.Version)
		if comparison < 0 {
			changeType = ChangeTypeDowngrade
		} else if comparison > 0 {
			changeType = ChangeTypeUpgrade
		} else {
			changeType = ChangeTypeUpdate
		}
	}

	displayTargetVersion := "(undefined)"
	if targetRelease != nil {
		displayTargetVersion = targetRelease.Spec.Version
	}

	sourceTag, err := opts.getReleaseGitTagFunc(sourceRelease)
	if err != nil {
		return nil, fmt.Errorf("getting tag for source version %s of release %s: %w", sourceRelease.Spec.Version, sourceRelease.Name, err)
	}

	targetTag := sourceTag
	if targetRelease != nil {
		targetTag, err = opts.getReleaseGitTagFunc(targetRelease)
		if err != nil {
			return nil, fmt.Errorf("getting tag for target version %s of release %s: %w", targetRelease.Spec.Version, targetRelease.Name, err)
		}
	}

	olderTag := targetTag
	newerTag := sourceTag
	if changeType == ChangeTypeDowngrade {
		olderTag = sourceTag
		newerTag = targetTag
	}

	repository := opts.getProjectRepositoryFunc(project)
	releaseInfo := ReleaseInfo{
		Name:         sourceRelease.Name,
		Project:      project,
		Repository:   repository,
		Source:       ReleaseWithGitTag{Release: sourceRelease, DisplayVersion: sourceRelease.Spec.Version, GitTag: sourceTag},
		Target:       ReleaseWithGitTag{Release: targetRelease, DisplayVersion: displayTargetVersion, GitTag: targetTag},
		OlderGitTag:  olderTag,
		NewerGitTag:  newerTag,
		IsUpgrade:    changeType == ChangeTypeUpgrade,
		ChangeType:   changeType,
		IsPrerelease: IsPrerelease(sourceRelease) || IsPrerelease(targetRelease),
	}

	if releaseInfo.IsPrerelease {
		return &releaseInfo, nil
	}

	projectDir, err := opts.getProjectSourceDirFunc(project)
	if err != nil {
		return nil, fmt.Errorf("getting project clone: %w", err)
	}

	commitsMetadata, err := opts.getCommitsMetadataFunc(projectDir, olderTag, newerTag)
	if err != nil {
		return nil, fmt.Errorf("getting commits metadata: %w", err)
	}

	gitHubAuthors, err := opts.getCommitsGitHubAuthorsFunc(project, olderTag, newerTag)
	if err != nil {
		return nil, fmt.Errorf("getting GitHub authors: %w", err)
	}

	codeOwners, err := opts.getCodeOwnersFunc(projectDir)
	if err != nil {
		return nil, fmt.Errorf("getting GitHub authors: %w", err)
	}
	releaseInfo.CodeOwners = codeOwners

	for _, metadata := range commitsMetadata {
		metadata.Message, err = injectPullRequestLinks(repository, metadata.Message)
		if err != nil {
			return nil, fmt.Errorf("injecting pull request links: %w", err)
		}

		shortMessage, _, _ := strings.Cut(metadata.Message, "\n")
		releaseInfo.Commits = append(releaseInfo.Commits, &CommitInfo{
			Sha:          metadata.Sha,
			ShortSha:     metadata.Sha[:min(7, len(metadata.Sha))],
			Author:       metadata.Author,
			GitHubAuthor: gitHubAuthors[metadata.Sha],
			Message:      metadata.Message,
			ShortMessage: shortMessage,
		})
	}

	return &releaseInfo, nil
}

func IsPrerelease(release *v1alpha1.Release) bool {
	if release == nil {
		return false
	}
	version := release.Spec.Version
	return semver.Prerelease("v"+version) != "" || semver.Build("v"+version) != ""
}

func renderMessage(messageTemplate string, info *PromotionInfo) (string, error) {
	getErrorMessage := func(err error) string {
		return fmt.Sprintf("Promote %d releases (%s -> %s)\nFailed to render message: %v",
			len(info.Releases), info.SourceEnvironment.Name,
			info.TargetEnvironment.Name, err)
	}

	if info.Error != nil {
		return getErrorMessage(info.Error), nil
	}

	tmpl, err := template.New("message").Funcs(sprig.FuncMap()).Parse(messageTemplate)
	if err != nil {
		return getErrorMessage(fmt.Errorf("parsing message template: %w", err)), nil
	}

	var message strings.Builder
	if err := tmpl.Execute(&message, info); err != nil {
		return getErrorMessage(fmt.Errorf("executing message template: %w", err)), nil
	}
	return message.String(), nil
}
