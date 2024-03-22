package promote

import (
	"cmp"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nestoca/joy/internal/links"

	"github.com/nestoca/joy/internal/info"

	"github.com/nestoca/joy/internal/style"

	"github.com/google/uuid"

	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/cross"
)

const (
	defaultCommitAndPRTemplate = `Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})`
)

type PerformOpts struct {
	list                cross.ReleaseList
	autoMerge           bool
	draft               bool
	dryRun              bool
	localOnly           bool
	commitTemplate      string
	pullRequestTemplate string
	infoProvider        info.Provider
	linksProvider       links.Provider
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

		p.PromptProvider.PrintUpdatingTargetRelease(targetEnv.Name, crossRelease.Name, promotedFile.Path, isCreatingTargetRelease)

		if opts.dryRun {
			fmt.Printf("ℹ️ Dry-run: skipping writing promoted release %s to: %s\n", style.Resource(crossRelease.Name), style.SecondaryInfo(promotedFile.Path))
		} else {
			if err := p.YamlWriter.WriteFile(promotedFile); err != nil {
				return "", fmt.Errorf("writing release %q promoted target yaml to file %q: %w", crossRelease.Name, promotedFile.Path, err)
			}
		}

		fmt.Printf("🧬 Collecting information about release %s...\n", style.Resource(crossRelease.Name))
		releaseInfo, err := getReleaseInfo(sourceRelease, targetRelease, opts)
		if err != nil {
			err = fmt.Errorf("collecting release %q info: %w", sourceRelease.Name, err)
			fmt.Printf("⚠️ %v\n", err)
			releaseInfo = &ReleaseInfo{
				Name:  sourceRelease.Name,
				Error: err,
			}
		}

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

	modeName := "normal"
	if opts.dryRun {
		modeName = "Dry-Run"
	}
	if opts.localOnly {
		modeName = "Local-Only"
	}

	branchName := getBranchName(info)
	if opts.dryRun || opts.localOnly {
		fmt.Printf("ℹ️ %s: skipping creation of branch %s\nFiles:\n%s\nCommit message:\n%s\n",
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

	if opts.dryRun || opts.localOnly {
		fmt.Printf("ℹ️ %s: skipping creation of pull request:\n%s\n%s\nReviewers:\n%s\nLabels:\n%s\n",
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
		if reviewer == "*" {
			continue
		}
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
