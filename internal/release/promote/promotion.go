package promote

import (
	"fmt"
	"strings"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

type Promotion struct {
	PromptProvider      PromptProvider
	GitProvider         GitProvider
	PullRequestProvider pr.PullRequestProvider
	YamlWriter          yml.Writer
	CommitTemplate      string
	PullRequestTemplate string
	InfoProvider        info.Provider
	LinksProvider       links.Provider

	// Prompt is the prompt to use for user interaction.
	promptProvider PromptProvider

	// Committer allows committing and pushing changes to git.
	gitProvider GitProvider

	// PullRequestProvider is the provider of pull requests.
	pullRequestProvider pr.PullRequestProvider

	commitTemplate      string
	pullRequestTemplate string
}

func NewPromotion(prompt PromptProvider, gitProvider GitProvider, pullRequestProvider pr.PullRequestProvider, yamlWriter yml.Writer, commitTemplate string, pullRequestTemplate string, getProjectRepositoryFunc func(proj *v1alpha1.Project) string, getProjectSourceDirFunc func(proj *v1alpha1.Project) (string, error), getCommitsMetadataFunc func(projectDir string, from string, to string) ([]*info.CommitMetadata, error), getCommitsGitHubAuthorsFunc func(proj *v1alpha1.Project, fromTag string, toTag string) (map[string]string, error), getReleaseGitTagFunc func(release *v1alpha1.Release) (string, error), getCodeOwnersFunc func(projectDir string) ([]string, error)) *Promotion {
	return &Promotion{
		promptProvider:      prompt,
		gitProvider:         gitProvider,
		pullRequestProvider: pullRequestProvider,
		YamlWriter:          yamlWriter,
		commitTemplate:      commitTemplate,
		pullRequestTemplate: pullRequestTemplate,
	}
}

type Opts struct {
	// Catalog contains candidate environments and releases to promote.
	Catalog *catalog.Catalog

	// SourceEnv is the source environment to promote from.
	SourceEnv *v1alpha1.Environment

	// TargetEnv is the target environment to promote to.
	TargetEnv *v1alpha1.Environment

	// Releases are the already selected releases.
	// If there is more than one, no need to prompt the user to select releases.
	Releases []string

	ReleasesFiltered bool

	// NoPrompt means that the Promote function should avoid interactive prompts at all costs.
	NoPrompt bool

	// AutoMerge indicates if PR created needs the auto-merge label
	AutoMerge bool

	// Draft indicates if PR created needs to be draft
	Draft bool

	// SelectedEnvironments is the list of environments selected by the user interactively via `joy env select`.
	SelectedEnvironments []*v1alpha1.Environment

	// DryRun indicates if the promotion should be performed in dry-run mode
	DryRun bool

	// LocalOnly indicates if the promotion should only write the promotion changes to the working tree without creating a branch, commit or pull request.
	LocalOnly bool
}

// Promote prompts user to select source and target environments and releases to promote and creates a pull request,
// returning its URL if any.
func (p *Promotion) Promote(opts Opts) (string, error) {
	if opts.DryRun {
		fmt.Println("ℹ️ Dry-run mode enabled: No changes will be made.")
	}

	if opts.LocalOnly {
		fmt.Println("ℹ️ Local-only mode enabled: The local repo will be modified, but not committed. No pull request will be created.")
	}

	// Prompt user to select source environment
	if opts.SourceEnv == nil {
		sourceEnvs, err := getSourceEnvironments(opts.SelectedEnvironments)
		if err != nil {
			return "", err
		}
		opts.SourceEnv, err = p.PromptProvider.SelectSourceEnvironment(sourceEnvs)
		if err != nil {
			return "", err
		}
	}

	// Prompt user to select target environment
	if opts.TargetEnv == nil {
		targetEnvs, err := getTargetEnvironments(opts.SelectedEnvironments, opts.SourceEnv)
		if err != nil {
			return "", err
		}
		opts.TargetEnv, err = p.PromptProvider.SelectTargetEnvironment(targetEnvs)
		if err != nil {
			return "", err
		}
	}

	if !opts.TargetEnv.Spec.Promotion.AllowAutoMerge && opts.AutoMerge {
		return "", fmt.Errorf("auto-merge is not allowed for target environment %s", opts.TargetEnv.Name)
	}

	// Validate promotability (only relevant if either or both environments were specified via command line flags)
	if !opts.SourceEnv.IsPromotableTo(opts.TargetEnv) {
		return "", fmt.Errorf("environment %s is not promotable to %s", opts.SourceEnv.Name, opts.TargetEnv.Name)
	}

	list, err := opts.Catalog.Releases.GetReleasesForPromotion(opts.SourceEnv, opts.TargetEnv)
	if err != nil {
		return "", fmt.Errorf("getting releases for promotion: %w", err)
	}

	if !list.HasAnyPromotableReleases() {
		p.PromptProvider.PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
		return "", nil
	}

	selectedList, err := func() (cross.ReleaseList, error) {
		if len(opts.Releases) > 0 {
			return list.OnlySpecificReleases(opts.Releases), nil
		}
		return p.PromptProvider.SelectReleases(list)
	}()
	if err != nil {
		return "", fmt.Errorf("selecting releases to promote: %w", err)
	}

	invalidList := selectedList.GetNonPromotableReleases(opts.SourceEnv, opts.TargetEnv)
	if len(invalidList) != 0 {
		invalid := strings.Join(invalidList, ", ")
		p.PromptProvider.PrintSelectedNonPromotableReleases(invalid, opts.TargetEnv.Name)
		return "", fmt.Errorf("cannot promote releases with non-standard version to %s environment", opts.TargetEnv.Name)
	}

	if !opts.NoPrompt {
		if err := p.preview(selectedList); err != nil {
			return "", fmt.Errorf("previewing: %w", err)
		}
	}

	// There's a previous check so only one option can be true at a time
	performParams := PerformOpts{
		list:                selectedList,
		autoMerge:           opts.AutoMerge,
		draft:               opts.Draft,
		dryRun:              opts.DryRun,
		localOnly:           opts.LocalOnly,
		commitTemplate:      p.CommitTemplate,
		pullRequestTemplate: p.PullRequestTemplate,
		infoProvider:        p.InfoProvider,
		linksProvider:       p.LinksProvider,
	}

	if opts.NoPrompt {
		return p.perform(performParams)
	}

	if opts.AutoMerge || opts.Draft {
		confirmed, err := p.PromptProvider.ConfirmCreatingPromotionPullRequest(opts.AutoMerge, opts.Draft)
		if err != nil {
			return "", fmt.Errorf("confirming creating promotion pull request: %w", err)
		}
		if !confirmed {
			p.PromptProvider.PrintCanceled()
			return "", nil
		}

		return p.perform(performParams)
	}

	// Prompt user to select creating a pull request
	answer, err := p.PromptProvider.SelectCreatingPromotionPullRequest()
	if err != nil {
		return "", fmt.Errorf("selecting create promotion pull request: %w", err)
	}

	switch answer {
	case Draft:
		performParams.draft = true
	case Cancel:
		p.PromptProvider.PrintCanceled()
		return "", nil
	}

	return p.perform(performParams)
}

func (p *Promotion) preview(list cross.ReleaseList) error {
	p.PromptProvider.PrintStartPreview()
	targetEnv := list.Environments[1]

	for _, rel := range list.Items {
		// Skip releases that are already in sync
		if rel.PromotedFile == nil {
			continue
		}

		targetRelease := rel.Releases[1]
		var targetReleaseFile *yml.File
		if targetRelease != nil {
			targetReleaseFile = targetRelease.File
		}
		err := p.PromptProvider.PrintReleasePreview(targetEnv.Name, rel.Name, targetReleaseFile, rel.PromotedFile)
		if err != nil {
			return fmt.Errorf("printing release preview: %w", err)
		}
	}

	p.PromptProvider.PrintEndPreview()
	return nil
}

func getSourceEnvironments(environments []*v1alpha1.Environment) ([]*v1alpha1.Environment, error) {
	envsMap := make(map[string]bool)
	for _, env := range environments {
		for _, source := range env.Spec.Promotion.FromEnvironments {
			envsMap[source] = true
		}
	}
	var envs []*v1alpha1.Environment
	for _, env := range environments {
		if envsMap[env.Name] {
			envs = append(envs, env)
		}
	}
	if len(envs) == 0 {
		return nil, fmt.Errorf("no promotable source environments found")
	}
	return envs, nil
}

func getTargetEnvironments(environments []*v1alpha1.Environment, sourceEnvironment *v1alpha1.Environment) ([]*v1alpha1.Environment, error) {
	var envs []*v1alpha1.Environment
	for _, env := range environments {
		if env.Name != sourceEnvironment.Name {
			for _, source := range env.Spec.Promotion.FromEnvironments {
				if source == sourceEnvironment.Name {
					envs = append(envs, env)
				}
			}
		}
	}
	if len(envs) == 0 {
		return nil, fmt.Errorf("no target environments found to promote from %s", sourceEnvironment.Name)
	}
	return envs, nil
}
