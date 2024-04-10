package promote

import (
	"fmt"
	"io"
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
	TemplateVariables   map[string]string
	InfoProvider        info.Provider
	LinksProvider       links.Provider
	Out                 io.Writer
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

	// All tells the promotion to select all available releases
	All bool

	// Omit is the list of releases to be omitted from selection, useful for combining with all
	Omit []string

	// KeepPrerelease indicates that prereleases in target environments must not be promoted to
	KeepPrerelease bool

	// Draft indicates if PR created needs to be draft
	Draft bool

	// SelectedEnvironments is the list of environments selected by the user interactively via `joy env select`.
	SelectedEnvironments []*v1alpha1.Environment

	// DryRun indicates if the promotion should be performed in dry-run mode
	DryRun bool

	// LocalOnly indicates if the promotion should only write the promotion changes to the working tree without creating a branch, commit or pull request.
	LocalOnly bool

	MaxColumnWidth int
}

// Promote prompts user to select source and target environments and releases to promote and creates a pull request,
// returning its URL if any.
func (p *Promotion) Promote(opts Opts) (string, error) {
	if opts.DryRun {
		p.println("ℹ️ Dry-run mode enabled: No changes will be made.")
	}

	if opts.LocalOnly {
		p.println("ℹ️ Local-only mode enabled: The local repo will be modified, but not committed. No pull request will be created.")
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

	selectedList, err := func() (cross.ReleaseList, error) {
		if opts.All {
			return list, nil
		}
		if len(opts.Releases) > 0 {
			return list.OnlySpecificReleases(opts.Releases)
		}
		return p.PromptProvider.SelectReleases(list, opts.MaxColumnWidth)
	}()
	if err != nil {
		return "", fmt.Errorf("selecting releases to promote: %w", err)
	}

	selectedList, err = selectedList.RemoveReleasesByName(opts.Omit)
	if err != nil {
		return "", fmt.Errorf("omitting releases: %w", err)
	}

	if opts.KeepPrerelease {
		selectedList = selectedList.Filter(func(release *cross.Release) bool {
			return len(release.Releases) != 2 || release.Releases[1] == nil || !IsPrerelease(release.Releases[1])
		})
	}

	if !selectedList.HasAnyPromotableReleases() {
		p.PromptProvider.PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
		return "", nil
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
		templateVariables:   p.TemplateVariables,
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

func (p *Promotion) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(p.Out, format, args...)
}

func (p *Promotion) println(a ...any) {
	_, _ = fmt.Fprintln(p.Out, a...)
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
