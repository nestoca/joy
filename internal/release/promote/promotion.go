package promote

import (
	"fmt"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/git/pr/github"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

type Promotion struct {
	// Prompt is the prompt to use for user interaction.
	promptProvider PromptProvider

	// Committer allows committing and pushing changes to git.
	gitProvider GitProvider

	// PullRequestProvider is the provider of pull requests.
	pullRequestProvider pr.PullRequestProvider

	// YamlWriter is the writer of YAML files.
	yamlWriter YamlWriter
}

func NewPromotion(prompt PromptProvider, gitProvider GitProvider, pullRequestProvider pr.PullRequestProvider, yamlWriter YamlWriter) *Promotion {
	return &Promotion{
		promptProvider:      prompt,
		gitProvider:         gitProvider,
		pullRequestProvider: pullRequestProvider,
		yamlWriter:          yamlWriter,
	}
}

func NewDefaultPromotion(catalogDir string) *Promotion {
	return NewPromotion(
		&InteractivePromptProvider{},
		NewShellGitProvider(catalogDir),
		github.NewPullRequestProvider(catalogDir),
		&FileSystemYamlWriter{},
	)
}

type Opts struct {
	// Catalog contains candidate environments and releases to promote.
	Catalog *catalog.Catalog

	// SourceEnv is the source environment to promote from.
	SourceEnv *v1alpha1.Environment

	// TargetEnv is the target environment to promote to.
	TargetEnv *v1alpha1.Environment

	// ReleasesFiltered indicates whether releases were filtered out by the user via command line flag or interactive selection.
	ReleasesFiltered bool

	// SelectedEnvironments is the list of environments selected by the user interactively via `joy env select`.
	SelectedEnvironments []*v1alpha1.Environment
}

// Promote prompts user to select source and target environments and releases to promote and creates a pull request,
// returning its URL if any.
func (p *Promotion) Promote(opts Opts) (string, error) {
	err := p.gitProvider.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return "", err
	}

	// Prompt user to select source environment
	if opts.SourceEnv == nil {
		sourceEnvs, err := getSourceEnvironments(opts.SelectedEnvironments)
		if err != nil {
			return "", err
		}
		opts.SourceEnv, err = p.promptProvider.SelectSourceEnvironment(sourceEnvs)
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
		opts.TargetEnv, err = p.promptProvider.SelectTargetEnvironment(targetEnvs)
		if err != nil {
			return "", err
		}
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
		p.promptProvider.PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
		return "", nil
	}

	list, err = p.promptProvider.SelectReleases(list)
	if err != nil {
		return "", fmt.Errorf("selecting releases to promote: %w", err)
	}
	if !list.HasAnyPromotableReleases() {
		p.promptProvider.PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
		return "", nil
	}

	err = p.preview(list)
	if err != nil {
		return "", fmt.Errorf("previewing: %w", err)
	}

	confirmed, err := p.promptProvider.ConfirmCreatingPromotionPullRequest()
	if err != nil {
		return "", fmt.Errorf("confirming creating promotion pull request: %w", err)
	}
	if !confirmed {
		p.promptProvider.PrintCanceled()
		return "", nil
	}

	prURL, err := p.perform(list)
	if err != nil {
		return "", fmt.Errorf("applying: %w", err)
	}
	return prURL, nil
}

func (p *Promotion) preview(list *cross.ReleaseList) error {
	p.promptProvider.PrintStartPreview()
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
		err := p.promptProvider.PrintReleasePreview(targetEnv.Name, rel.Name, targetReleaseFile, rel.PromotedFile)
		if err != nil {
			return fmt.Errorf("printing release preview: %w", err)
		}
	}

	p.promptProvider.PrintEndPreview()
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
