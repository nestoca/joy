package promote

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/style"
)

const (
	defaultCommitAndPRTemplate = `Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})`

	// RenderingFailedLabel marks promotion PRs whose message could not be fully rendered.
	RenderingFailedLabel = "rendering:failed"
)

type PerformOpts struct {
	list                cross.ReleaseList
	autoMerge           bool
	draft               bool
	dryRun              bool
	localOnly           bool
	commitTemplate      string
	pullRequestTemplate string
	templateVariables   map[string]string
	infoProvider        info.Provider
	linksProvider       links.Provider
	reviewers           []string
	updatePullRequest   *pr.PullRequest
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
		Variables:         opts.templateVariables,
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

		p.PromptProvider.PrintUpdatingTargetRelease(targetEnv.Name, crossRelease.Name, promotedFile.Path, isCreatingTargetRelease)

		if opts.dryRun {
			p.printf("ℹ️ Dry-run: skipping writing promoted release %s to: %s\n", style.Resource(crossRelease.Name), style.SecondaryInfo(promotedFile.Path))
		} else if opts.updatePullRequest != nil {
			p.printf("ℹ️ Updating existing pull request: skipping writing promoted release %s to: %s\n", style.Resource(crossRelease.Name), style.SecondaryInfo(promotedFile.Path))
		} else {
			if err := p.YamlWriter.WriteFile(promotedFile); err != nil {
				return "", fmt.Errorf("writing release %q promoted target yaml to file %q: %w", crossRelease.Name, promotedFile.Path, err)
			}
		}

		if opts.localOnly {
			continue
		}

		p.printf("🧬 Collecting information about release %s...\n", style.Resource(crossRelease.Name))
		releaseInfo, err := getReleaseInfo(crossRelease, sourceRelease, targetRelease, opts)
		if err != nil {
			err = fmt.Errorf("collecting release %q info: %w", sourceRelease.Name, err)
			p.printf("⚠️ %v\n", err)
			releaseInfo = &ReleaseInfo{
				Name:  sourceRelease.Name,
				Error: err,
			}
		}

		info.Releases = append(info.Releases, releaseInfo)
		info.Error = errors.Join(info.Error, releaseInfo.Error)
	}

	if len(promotedFiles) == 0 {
		p.println("🤷 Nothing to promote!")
		return "", nil
	}

	if opts.localOnly {
		return "", nil
	}

	commitTemplate := cmp.Or(opts.commitTemplate, defaultCommitAndPRTemplate)
	commitMessage, err := renderMessage(commitTemplate, info)
	if err != nil {
		return "", fmt.Errorf("rendering commit message: %w", err)
	}

	modeName := "normal"
	if opts.dryRun {
		modeName = "Dry-Run"
	}
	if opts.localOnly {
		modeName = "Local-Only"
	}

	branchName := getBranchName(info)
	if opts.updatePullRequest != nil {
		branchName = opts.updatePullRequest.Branch
		p.printf("ℹ️ Updating existing pull request: skipping creation of branch %s\n", style.Resource(branchName))
	} else if opts.dryRun || opts.localOnly {
		p.printf("ℹ️ %s: skipping creation of branch %s\nFiles:\n%s\nCommit message:\n%s\n",
			modeName,
			style.Resource(branchName), style.SecondaryInfo("- "+strings.Join(promotedFiles, "\n- ")),
			style.SecondaryInfo(commitMessage))
	} else {
		err = p.GitProvider.CreateAndPushBranchWithFiles(branchName, promotedFiles, commitMessage)
		if err != nil {
			return "", err
		}
		p.PromptProvider.PrintBranchCreated(branchName, commitMessage)
	}

	var labels []string
	labels = append(labels, "environment:"+info.TargetEnvironment.Name)
	for _, release := range info.Releases {
		labels = append(labels, "release:"+release.Name)
	}

	if opts.autoMerge {
		labels = append(labels, "auto-merge")
	}

	if info.Error != nil {
		labels = append(labels, RenderingFailedLabel)
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

	inferredReviewers := getReviewers(info)
	reviewers := MergeUnique(inferredReviewers, opts.reviewers)

	prBody = AppendPromotionMetadata(prBody, PromotionMetadata{
		Source:    sourceEnv.Name,
		Target:    targetEnv.Name,
		Releases:  getReleaseNames(info),
		Variables: opts.templateVariables,
		Reviewers: opts.reviewers,
	})

	if opts.updatePullRequest != nil {
		return "", p.updatePullRequest(opts.updatePullRequest, prTitle, prBody, reviewers, info.Error, opts.dryRun)
	}

	if opts.dryRun || opts.localOnly {
		p.printf("ℹ️ %s: skipping creation of pull request:\n%s\n%s\nReviewers:\n%s\nLabels:\n%s\n",
			modeName,
			style.SecondaryInfo(prTitle), style.SecondaryInfo(prBody),
			style.SecondaryInfo("- "+strings.Join(reviewers, "\n- ")),
			style.SecondaryInfo("- "+strings.Join(labels, "\n- ")))
		p.PromptProvider.PrintCompleted()
		return "", nil
	}

	prURL, err := p.PullRequestProvider.Create(pr.CreateParams{
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
		p.PromptProvider.PrintDraftPullRequestCreated(prURL)
	} else {
		p.PromptProvider.PrintPullRequestCreated(prURL)
	}

	if err := p.GitProvider.CheckoutMasterBranch(); err != nil {
		return "", fmt.Errorf("checking out master: %w", err)
	}

	p.PromptProvider.PrintCompleted()

	return prURL, nil
}

// updatePullRequest applies the re-rendered title, body, reviewers and rendering label to an existing pull request,
// leaving it untouched when nothing changed.
func (p *Promotion) updatePullRequest(current *pr.PullRequest, title, body string, reviewers []string, renderErr error, dryRun bool) error {
	if renderErr != nil && !slices.Contains(current.Labels, RenderingFailedLabel) {
		return fmt.Errorf("re-rendering pull request %s failed, leaving it unchanged: %w", current.Link(), renderErr)
	}

	params := pr.UpdateParams{Number: current.Number}
	if strings.TrimSpace(title) != strings.TrimSpace(current.Title) {
		params.Title = title
	}
	if strings.TrimSpace(body) != strings.TrimSpace(current.Body) {
		params.Body = body
	}
	for _, reviewer := range reviewers {
		if pr.IsRequestableReviewer(reviewer) && !slices.ContainsFunc(current.Reviewers, func(existing string) bool { return strings.EqualFold(existing, reviewer) }) {
			params.AddReviewers = append(params.AddReviewers, reviewer)
		}
	}
	if renderErr == nil && slices.Contains(current.Labels, RenderingFailedLabel) {
		params.RemoveLabels = []string{RenderingFailedLabel}
	}

	if params.IsEmpty() {
		p.printf("✅ Pull request %s is already up to date\n", style.Link(current.Link()))
	} else if dryRun {
		p.printf("ℹ️ Dry-run: skipping update of pull request %s:\nTitle: %s\nBody:\n%s\nAdd reviewers: %s\nRemove labels: %s\n",
			style.Link(current.Link()),
			style.SecondaryInfo(cmp.Or(params.Title, "(unchanged)")),
			style.SecondaryInfo(cmp.Or(params.Body, "(unchanged)")),
			style.SecondaryInfo(strings.Join(params.AddReviewers, ", ")),
			style.SecondaryInfo(strings.Join(params.RemoveLabels, ", ")))
	} else {
		if err := p.PullRequestProvider.Update(params); err != nil {
			return fmt.Errorf("updating pull request: %w", err)
		}
		p.PromptProvider.PrintPullRequestUpdated(current.Link())
	}
	p.PromptProvider.PrintCompleted()

	if renderErr != nil {
		return fmt.Errorf("pull request %s still has rendering errors: %w", current.Link(), renderErr)
	}
	return nil
}

func getReleaseNames(info *PromotionInfo) []string {
	names := make([]string, 0, len(info.Releases))
	for _, release := range info.Releases {
		names = append(names, release.Name)
	}
	return names
}

func getReviewers(info *PromotionInfo) []string {
	uniqueAuthors := make(map[string]struct{})
	for _, release := range info.Releases {
		for _, owner := range release.Reviewers {
			uniqueAuthors[owner] = struct{}{}
		}

		// releaseInfo.Project may be nil if an error was encountered.
		// therefore we need to check if the project exists before dereferencing it
		if release.Project != nil {
			for _, owner := range release.Project.Spec.Reviewers {
				uniqueAuthors[owner] = struct{}{}
			}
		}

		for _, commit := range release.Commits {
			uniqueAuthors[commit.GitHubAuthor] = struct{}{}
		}
	}

	return slices.DeleteFunc(slices.Sorted(maps.Keys(uniqueAuthors)), func(reviewer string) bool {
		return !pr.IsRequestableReviewer(reviewer)
	})
}

func MergeUnique[T comparable](a, b []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(a)+len(b))

	for _, v := range append(a, b...) {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
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
