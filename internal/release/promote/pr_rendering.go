package promote

import (
	"fmt"
	"regexp"
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
	Source       EnvironmentReleaseInfo
	Target       EnvironmentReleaseInfo
	OlderGitTag  string
	NewerGitTag  string
	IsPrerelease bool
	ChangeType   ChangeType
	Commits      []*CommitInfo
	Error        error
}

type EnvironmentReleaseInfo struct {
	*v1alpha1.Release
	DisplayVersion string
	GitTag         string
	Links          map[string]string
}

type PromotionInfo struct {
	SourceEnvironment *v1alpha1.Environment
	TargetEnvironment *v1alpha1.Environment
	Releases          []*ReleaseInfo
	Variables         map[string]string
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

	sourceTag, err := opts.infoProvider.GetReleaseGitTag(sourceRelease)
	if err != nil {
		return nil, fmt.Errorf("getting tag for source version %s of release %s: %w", sourceRelease.Spec.Version, sourceRelease.Name, err)
	}

	targetTag := sourceTag
	if targetRelease != nil {
		targetTag, err = opts.infoProvider.GetReleaseGitTag(targetRelease)
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

	sourceLinks, err := opts.linksProvider.GetReleaseLinks(sourceRelease)
	if err != nil {
		return nil, fmt.Errorf("getting source release links: %w", err)
	}
	targetLinks, err := opts.linksProvider.GetReleaseLinks(targetRelease)
	if err != nil {
		return nil, fmt.Errorf("getting target release links: %w", err)
	}

	repository := opts.infoProvider.GetProjectRepository(project)
	releaseInfo := ReleaseInfo{
		Name:         sourceRelease.Name,
		Project:      project,
		Repository:   repository,
		Source:       EnvironmentReleaseInfo{Release: sourceRelease, DisplayVersion: sourceRelease.Spec.Version, GitTag: sourceTag, Links: sourceLinks},
		Target:       EnvironmentReleaseInfo{Release: targetRelease, DisplayVersion: displayTargetVersion, GitTag: targetTag, Links: targetLinks},
		OlderGitTag:  olderTag,
		NewerGitTag:  newerTag,
		ChangeType:   changeType,
		IsPrerelease: IsPrerelease(sourceRelease) || IsPrerelease(targetRelease),
	}

	if releaseInfo.IsPrerelease {
		return &releaseInfo, nil
	}

	projectDir, err := opts.infoProvider.GetProjectSourceDir(project)
	if err != nil {
		return nil, fmt.Errorf("getting project clone: %w", err)
	}

	commitsMetadata, err := opts.infoProvider.GetCommitsMetadata(projectDir, olderTag, newerTag)
	if err != nil {
		return nil, fmt.Errorf("getting commits metadata: %w", err)
	}

	gitHubAuthors, err := opts.infoProvider.GetCommitsGitHubAuthors(project, olderTag, newerTag)
	if err != nil {
		return nil, fmt.Errorf("getting GitHub authors: %w", err)
	}

	codeOwners, err := opts.infoProvider.GetCodeOwners(projectDir)
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

var pullRequestReferenceRegex = regexp.MustCompile(`(?m)(^|\s)#(\d+)\b`)

func injectPullRequestLinks(repo string, text string) (string, error) {
	// Iterate over the matches in reverse order, to prevent replacement from offsetting indexes
	matches := pullRequestReferenceRegex.FindAllStringSubmatchIndex(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		prefix := text[match[2]:match[3]]
		prNumber := text[match[4]:match[5]]
		replacement := fmt.Sprintf("[#%s](https://github.com/%s/pull/%s)", prNumber, repo, prNumber)
		text = text[:match[0]] + prefix + replacement + text[match[1]:]
	}

	return text, nil
}
