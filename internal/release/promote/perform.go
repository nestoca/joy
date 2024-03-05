package promote

import (
	"cmp"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/internal/style"

	"github.com/google/uuid"

	"github.com/Masterminds/sprig/v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/cross"
)

const (
	defaultCommitAndPRTemplate = `Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})`
)

type PerformOpts struct {
	list                        *cross.ReleaseList
	autoMerge                   bool
	draft                       bool
	dryRun                      bool
	commitTemplate              string
	pullRequestTemplate         string
	getProjectSourceDirFunc     func(proj *v1alpha1.Project) (string, error)
	getProjectRepositoryFunc    func(proj *v1alpha1.Project) string
	getCommitsMetadataFunc      func(projectDir, fromTag, toTag string) ([]*CommitMetadata, error)
	getCodeOwnersFunc           func(projectDir string) ([]string, error)
	getCommitsGitHubAuthorsFunc func(proj *v1alpha1.Project, fromTag, toTag string) (map[string]string, error)
	getReleaseGitTagFunc        func(release *v1alpha1.Release) (string, error)
}

type ReleaseWithGitTag struct {
	*v1alpha1.Release
	DisplayVersion string
	GitTag         string
}

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
	IsUpgrade    bool
	Commits      []*CommitInfo
	Error        error
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

		if opts.dryRun {
			fmt.Printf("ℹ️ Dry-run: skipping writing promoted release %s to: %s\n", style.Resource(crossRelease.Name), style.SecondaryInfo(promotedFile.Path))
		} else {
			if err := p.yamlWriter.Write(promotedFile); err != nil {
				return "", fmt.Errorf("writing release %q promoted target yaml to file %q: %w", crossRelease.Name, promotedFile.Path, err)
			}
		}

		fmt.Printf("🧬 Collecting information about release %s...\n", style.Resource(crossRelease.Name))
		releaseInfo := getReleaseInfo(sourceRelease, targetRelease, opts)

		info.Releases = append(info.Releases, releaseInfo)
		info.Error = errors.Join(info.Error, releaseInfo.Error)
	}

	if len(promotedFiles) == 0 {
		fmt.Println("🤷 Nothing to promote!")
		return "", nil
	}

	commitTemplate := cmp.Or(opts.commitTemplate, defaultCommitAndPRTemplate)
	commitMessage, err := renderMessage(commitTemplate, info)
	if err != nil {
		return "", fmt.Errorf("rendering commit message: %w", err)
	}

	branchName := getBranchName(info)
	if opts.dryRun {
		fmt.Printf("ℹ️ Dry-run: skipping creation of branch %s\nFiles:\n%s\nCommit message:\n%s\n",
			style.Resource(branchName), style.SecondaryInfo("- "+strings.Join(promotedFiles, "\n- ")),
			style.SecondaryInfo(commitMessage))
	} else {
		err = p.gitProvider.CreateAndPushBranchWithFiles(branchName, promotedFiles, commitMessage)
		if err != nil {
			return "", err
		}
		p.promptProvider.PrintBranchCreated(branchName, commitMessage)
	}

	var labels []string
	labels = append(labels, "environment:"+info.TargetEnvironment.Name)
	for _, release := range info.Releases {
		labels = append(labels, "release:"+release.Name)
	}

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

	reviewers := getReviewers(info)

	if opts.dryRun {
		fmt.Printf("ℹ️ Dry-run: skipping creation of pull request:\n%s\n%s\nReviewers:\n%s\nLabels:\n%s\n",
			style.SecondaryInfo(prTitle), style.SecondaryInfo(prBody),
			style.SecondaryInfo("- "+strings.Join(reviewers, "\n- ")),
			style.SecondaryInfo("- "+strings.Join(labels, "\n- ")))
		p.promptProvider.PrintCompleted()
		return "", nil
	}

	prURL, err := p.pullRequestProvider.Create(pr.CreateParams{
		Branch:    branchName,
		Title:     prTitle,
		Body:      prBody,
		Labels:    labels,
		Draft:     opts.draft,
		Reviewers: reviewers,
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
	uniqueAuthors := make(map[string]bool)
	for _, release := range info.Releases {
		for _, owner := range release.CodeOwners {
			uniqueAuthors[owner] = true
		}

		// releaseInfo.Project may be nil if an error was encountered.
		// therefore we need to check if the project exists before dereferencing it
		if release.Project != nil {
			for _, owner := range release.Project.Spec.CodeOwners {
				uniqueAuthors[owner] = true
			}
		}

		for _, commit := range release.Commits {
			uniqueAuthors[commit.GitHubAuthor] = true
		}
	}

	var reviewers []string
	for reviewer := range uniqueAuthors {
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
	getErrorMessage := func(err error) string {
		return fmt.Sprintf("Promote %d releases (%s -> %s)\nFailed to render message: %v",
			len(info.Releases), info.SourceEnvironment.Name,
			info.TargetEnvironment.Name, err)
	}

	if info.Error != nil {
		return getErrorMessage(fmt.Errorf("collecting promotion info: %w", info.Error)), nil
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

func getReleaseInfo(sourceRelease, targetRelease *v1alpha1.Release, opts PerformOpts) *ReleaseInfo {
	getAndPrintReleaseInfoWithError := func(msg string, args ...any) *ReleaseInfo {
		err := fmt.Errorf("Failed to get info for release "+sourceRelease.Name+": "+msg, args...)
		fmt.Printf("⚠️ %v\n", err)
		return &ReleaseInfo{
			Name:  sourceRelease.Name,
			Error: err,
		}
	}

	project := sourceRelease.Project
	isUpgrade := true
	if targetRelease != nil {
		isUpgrade = semver.Compare("v"+sourceRelease.Spec.Version, "v"+targetRelease.Spec.Version) > 0
	}

	isPrerelease := false
	if semver.Build("v"+sourceRelease.Spec.Version) != "" ||
		semver.Prerelease("v"+sourceRelease.Spec.Version) != "" ||
		(targetRelease != nil && (semver.Build("v"+targetRelease.Spec.Version) != "" || semver.Prerelease("v"+targetRelease.Spec.Version) != "")) {
		isPrerelease = true
	}

	displayTargetVersion := "(undefined)"
	if targetRelease != nil {
		displayTargetVersion = targetRelease.Spec.Version
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

	olderTag := targetTag
	newerTag := sourceTag
	if !isUpgrade {
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
		IsUpgrade:    isUpgrade,
		IsPrerelease: isPrerelease,
	}

	if isPrerelease {
		return &releaseInfo
	}

	projectDir, err := opts.getProjectSourceDirFunc(project)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting project clone: %w", err)
	}

	commitsMetadata, err := opts.getCommitsMetadataFunc(projectDir, olderTag, newerTag)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting commits metadata: %w", err)
	}

	gitHubAuthors, err := opts.getCommitsGitHubAuthorsFunc(project, olderTag, newerTag)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting GitHub authors: %w", err)
	}

	codeOwners, err := opts.getCodeOwnersFunc(projectDir)
	if err != nil {
		return getAndPrintReleaseInfoWithError("getting GitHub authors: %w", err)
	}
	releaseInfo.CodeOwners = codeOwners

	for _, metadata := range commitsMetadata {
		shortSha := metadata.Sha
		if len(shortSha) > 7 {
			shortSha = shortSha[:7]
		}

		metadata.Message, err = injectPullRequestLinks(repository, metadata.Message)
		if err != nil {
			return getAndPrintReleaseInfoWithError("injecting pull request links: %w", err)
		}

		shortMessage := metadata.Message
		if strings.Contains(shortMessage, "\n") {
			shortMessage = strings.SplitN(shortMessage, "\n", 2)[0]
		}

		gitHubAuthor := gitHubAuthors[metadata.Sha]

		releaseInfo.Commits = append(releaseInfo.Commits, &CommitInfo{
			Sha:          metadata.Sha,
			ShortSha:     shortSha,
			Author:       metadata.Author,
			GitHubAuthor: gitHubAuthor,
			Message:      metadata.Message,
			ShortMessage: shortMessage,
		})
	}

	return &releaseInfo
}
