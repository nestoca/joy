package promote

import (
	"fmt"
	"github.com/nestoca/joy/api/v1alpha1"
)

type Promotion struct {
	// GitBranchProvider is the provider for managing git branches.
	BranchProvider BranchProvider

	// PullRequestProvider is the provider of pull requests.
	PullRequestProvider PullRequestProvider

	// Prompt is the prompt to use for user interaction.
	Prompt PromptProvider
}

func NewPromotion(branchProvider BranchProvider, pullRequestProvider PullRequestProvider, prompt PromptProvider) *Promotion {
	return &Promotion{
		BranchProvider:      branchProvider,
		PullRequestProvider: pullRequestProvider,
		Prompt:              prompt,
	}
}

func NewDefaultPromotion() *Promotion {
	return NewPromotion(
		&GitBranchProvider{},
		&GitHubPullRequestProvider{},
		&InteractivePromptProvider{},
	)
}

// Promote prompts user to create a pull request for current branch and to select environment to auto-promote builds
// of pull request to, and then configures the pull request accordingly.
func (p *Promotion) Promote(environments []*v1alpha1.Environment) error {
	branch, err := p.BranchProvider.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if branch == "master" || branch == "main" {
		p.Prompt.PrintMasterBranchPromotion()
		return nil
	}

	exists, err := p.PullRequestProvider.Exists(branch)
	if err != nil {
		return fmt.Errorf("checking if pull request exists for branch %s: %w", branch, err)
	}

	if !exists {
		shouldCreate, err := p.Prompt.WhetherToCreateMissingPullRequest()
		if err != nil {
			return fmt.Errorf("prompting user to create pull request: %w", err)
		}
		if !shouldCreate {
			p.Prompt.PrintNotCreatingPullRequest()
			return nil
		}

		err = p.PullRequestProvider.CreateInteractively(branch)
		if err != nil {
			return fmt.Errorf("creating pull request: %w", err)
		}
	}

	env, err := p.PullRequestProvider.GetPromotionEnvironment(branch)
	if err != nil {
		return fmt.Errorf("getting promotion environment for branch %s pull request: %w", branch, err)
	}

	promotableEnvironmentNames := getPromotableEnvironmentNames(environments)
	env, err = p.Prompt.WhichEnvironmentToPromoteTo(promotableEnvironmentNames, env)
	if err != nil {
		return fmt.Errorf("prompting user to select promotion environment: %w", err)
	}

	if err := p.PullRequestProvider.SetPromotionEnvironment(branch, env); err != nil {
		return fmt.Errorf("setting promotion for branch %s pull request to %q environment: %w", branch, env, err)
	}
	if env != "" {
		p.Prompt.PrintPromotionConfigured(branch, env)
	} else {
		p.Prompt.PrintPromotionDisabled(branch)
	}
	return nil
}

func getPromotableEnvironmentNames(environments []*v1alpha1.Environment) []string {
	var names []string
	for _, env := range environments {
		if env.Spec.Promotion.FromPullRequests {
			names = append(names, env.Name)
		}
	}
	return names
}
