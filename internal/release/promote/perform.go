package promote

import (
	"cmp"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/google/uuid"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/cross"
)

const (
	defaultCommitAndPRTemplate = `Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})`
)

type PerformOpts struct {
	list                      *cross.ReleaseList
	autoMerge                 bool
	draft                     bool
	commitTemplate            string
	pullRequestTemplate       string
	getProjectSourceDirFunc   func(proj *v1alpha1.Project) (string, error)
	getProjectRepositoryFunc  func(proj *v1alpha1.Project) string
	getCommitsMetadataFunc    func(projectDir, from, to string) ([]*CommitMetadata, error)
	getCommitGitHubAuthorFunc func(proj *v1alpha1.Project, sha string) (string, error)
	getReleaseGitTagFunc      func(release *v1alpha1.Release) (string, error)
}

type ReleaseWithGitTag struct {
	*v1alpha1.Release
	DisplayVersion string
	GitTag         string
}

type ReleaseInfo struct {
	Name       string
	Project    *v1alpha1.Project
	Repository string
	Source     ReleaseWithGitTag
	Target     ReleaseWithGitTag
	Commits    []*CommitInfo
	Error      error
}

type PromotionInfo struct {
	SourceEnvironment *v1alpha1.Environment
	TargetEnvironment *v1alpha1.Environment
	Releases          []*ReleaseInfo
}

type CommitInfo struct {
	Sha          string
	ShortSha     string
	Author       string
	GitHubAuthor string
	Message      string
}

// perform performs the promotion of all releases in given list and returns PR url if any
func (p *Promotion) perform(opts PerformOpts) (string, error) {
	if len(opts.list.Environments) != 2 {
		return "", fmt.Errorf("expecting 2 environments, got %d", len(opts.list.Environments))
	}

	sourceEnv := opts.list.Environments[0]
	targetEnv := opts.list.Environments[1]

	info := &PromotionInfo{
		SourceEnvironment: sourceEnv,
		TargetEnvironment: targetEnv,
	}

	var promotedFiles []string
	for _, crossRelease := range opts.list.SortedCrossReleases() {
		promotedFile := crossRelease.PromotedFile
		if promotedFile == nil {
			continue
		}
		promotedFiles = append(promotedFiles, promotedFile.Path)

		sourceRelease := crossRelease.Releases[0]
		targetRelease := crossRelease.Releases[1]
		isCreatingTargetRelease := targetRelease == nil

		p.promptProvider.PrintUpdatingTargetRelease(targetEnv.Name, crossRelease.Name, promotedFile.Path, isCreatingTargetRelease)

		if err := p.yamlWriter.Write(promotedFile); err != nil {
			return "", fmt.Errorf("writing release %q promoted target yaml to file %q: %w", crossRelease.Name, promotedFile.Path, err)
		}

		releaseInfo := getReleaseInfo(sourceRelease, targetRelease, opts)
		info.Releases = append(info.Releases, releaseInfo)
	}

	if len(promotedFiles) == 0 {
		return "", fmt.Errorf("no releases promoted, should not reach this point")
	}

	commitTemplate := cmp.Or(opts.commitTemplate, defaultCommitAndPRTemplate)
	message, err := renderMessage(commitTemplate, info)
	if err != nil {
		return "", fmt.Errorf("rendering commit message: %w", err)
	}

	branchName := getBranchName(info)
	err = p.gitProvider.CreateAndPushBranchWithFiles(branchName, promotedFiles, message)
	if err != nil {
		return "", err
	}
	p.promptProvider.PrintBranchCreated(branchName, message)

	var labels []string
	if opts.autoMerge {
		labels = append(labels, "auto-merge")
	}

	pullRequestTemplate := cmp.Or(opts.pullRequestTemplate, defaultCommitAndPRTemplate)
	prMessage, err := renderMessage(pullRequestTemplate, info)
	if err != nil {
		return "", fmt.Errorf("rendering pull request message: %w", err)
	}
	prLines := strings.SplitN(prMessage, "\n", 2)
	prTitle := prLines[0]
	prBody := ""
	if len(prLines) > 1 {
		prBody = prLines[1]
	}

	prURL, err := p.pullRequestProvider.Create(pr.CreateParams{
		Branch:    branchName,
		Title:     prTitle,
		Body:      prBody,
		Labels:    labels,
		Draft:     opts.draft,
		Reviewers: getReviewers(info),
	})
	if err != nil {
		return "", fmt.Errorf("creating pull request: %w", err)
	}

	if opts.draft {
		p.promptProvider.PrintDraftPullRequestCreated(prURL)
	} else {
		p.promptProvider.PrintPullRequestCreated(prURL)
	}

	if err := p.gitProvider.CheckoutMasterBranch(); err != nil {
		return "", fmt.Errorf("checking out master: %w", err)
	}

	p.promptProvider.PrintCompleted()

	return prURL, nil
}

func getReviewers(info *PromotionInfo) []string {
	uniqueAuthors := make(map[string]string)
	for _, release := range info.Releases {
		for _, commit := range release.Commits {
			uniqueAuthors[commit.GitHubAuthor] = commit.GitHubAuthor
		}
	}

	var reviewers []string
	for _, reviewer := range uniqueAuthors {
		reviewers = append(reviewers, reviewer)
	}
	sort.Strings(reviewers)
	return reviewers
}

func getBranchName(info *PromotionInfo) string {
	var releases string
	if len(info.Releases) == 1 {
		releases = info.Releases[0].Name
	} else {
		releases = fmt.Sprintf("%d-releases", len(info.Releases))
	}
	uniqueID := uuid.New().String()
	name := fmt.Sprintf("promote-%s-from-%s-to-%s-%s", releases, info.SourceEnvironment.Name, info.TargetEnvironment.Name, uniqueID)
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

func renderMessage(messageTemplate string, info *PromotionInfo) (string, error) {
	tmpl, err := template.New("message").Parse(messageTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing message template: %w", err)
	}

	var message strings.Builder
	if err := tmpl.Execute(&message, info); err != nil {
		return "", fmt.Errorf("executing message template: %w", err)
	}
	return message.String(), nil
}

func getReleaseInfo(sourceRelease, targetRelease *v1alpha1.Release, opts PerformOpts) *ReleaseInfo {
	getAndPrintReleaseInfoWithError := func(msg string, args ...any) *ReleaseInfo {
		err := fmt.Errorf(msg, args...)
		fmt.Printf("⚠️ Failed to get info for release %s: %v\n", sourceRelease.Name, err)
		return &ReleaseInfo{
			Name:  sourceRelease.Name,
			Error: err,
		}
	}

	projectDir, err := opts.getProjectSourceDirFunc(sourceRelease.Project)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting project clone: %w", err)
	}

	sourceTag, err := opts.getReleaseGitTagFunc(sourceRelease)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting tag for source version %s of release %s: %w", sourceRelease.Spec.Version, sourceRelease.Name, err)
	}

	targetTag := sourceTag
	if targetRelease != nil {
		targetTag, err = opts.getReleaseGitTagFunc(targetRelease)
		if err != nil {
			return getAndPrintReleaseInfoWithError("getting tag for target version %s of release %s: %w", targetRelease.Spec.Version, targetRelease.Name, err)
		}
	}

	commitsMetadata, err := opts.getCommitsMetadataFunc(projectDir, sourceTag, targetTag)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting commits metadata from %q: %w", projectDir, err)
	}

	var commits []*CommitInfo
	for _, metadata := range commitsMetadata {
		gitHubAuthor, err := opts.getCommitGitHubAuthorFunc(sourceRelease.Project, metadata.Sha)
		if err != nil {
			return getAndPrintReleaseInfoWithError("getting GitHub author for project %s commit %s: %w", sourceRelease.Project.Name, metadata.Sha, err)
		}

		shortSha := metadata.Sha
		if len(shortSha) > 7 {
			shortSha = shortSha[:7]
		}
		commits = append(commits, &CommitInfo{
			Sha:          metadata.Sha,
			ShortSha:     shortSha,
			Author:       metadata.Author,
			GitHubAuthor: gitHubAuthor,
			Message:      metadata.Message,
		})
	}

	displayTargetVersion := "(undefined)"
	if targetRelease != nil {
		displayTargetVersion = targetRelease.Spec.Version
	}

	return &ReleaseInfo{
		Name:       sourceRelease.Name,
		Project:    sourceRelease.Project,
		Repository: opts.getProjectRepositoryFunc(sourceRelease.Project),
		Source:     ReleaseWithGitTag{Release: sourceRelease, DisplayVersion: sourceRelease.Spec.Version, GitTag: sourceTag},
		Target:     ReleaseWithGitTag{Release: targetRelease, DisplayVersion: displayTargetVersion, GitTag: targetTag},
		Commits:    commits,
	}
}
