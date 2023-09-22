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
	PromptProvider PromptProvider
}

func NewPromotion(branchProvider BranchProvider, pullRequestProvider PullRequestProvider, prompt PromptProvider) *Promotion {
	return &Promotion{
		BranchProvider:      branchProvider,
		PullRequestProvider: pullRequestProvider,
		PromptProvider:      prompt,
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
	if err := p.PullRequestProvider.EnsureInstalledAndAuthorized(); err != nil {
		return nil
	}

	branch, err := p.BranchProvider.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if branch == "master" || branch == "main" {
		p.PromptProvider.PrintBranchDoesNotSupportAutoPromotion(branch)
		return nil
	}

	exists, err := p.PullRequestProvider.Exists(branch)
	if err != nil {
		return fmt.Errorf("checking if pull request exists for branch %s: %w", branch, err)
	}

	if !exists {
		shouldCreate, err := p.PromptProvider.WhetherToCreateMissingPullRequest()
		if err != nil {
			return fmt.Errorf("prompting user to create pull request: %w", err)
		}
		if !shouldCreate {
			p.PromptProvider.PrintNotCreatingPullRequest()
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
	env, err = p.PromptProvider.WhichEnvironmentToPromoteTo(promotableEnvironmentNames, env)
	if err != nil {
		return fmt.Errorf("prompting user to select promotion environment: %w", err)
	}

	// Disable auto-promotion on other pull requests promoting to same environment
	branchesPromotingToEnv, err := p.PullRequestProvider.GetBranchesPromotingToEnvironment(env)
	if err != nil {
		return fmt.Errorf("getting branches configured for auto-promotion to %q environment: %w", env, err)
	}
	for _, branchPromotingToEnv := range branchesPromotingToEnv {
		if branchPromotingToEnv == branch {
			p.PromptProvider.PrintPromotionAlreadyConfigured(branch, env)
			return nil
		}
		shouldDisable, err := p.PromptProvider.ConfirmDisablingPromotionOnOtherPullRequest(branchPromotingToEnv, env)
		if err != nil {
			return fmt.Errorf("prompting user to confirm disabling promotion on other pull request: %w", err)
		}
		if shouldDisable {
			if err := p.PullRequestProvider.SetPromotionEnvironment(branchPromotingToEnv, ""); err != nil {
				return fmt.Errorf("disabling promotion for branch %s pull request: %w", branchPromotingToEnv, err)
			}
		} else {
			p.PromptProvider.PrintPromotionNotConfigured(branch, env)
			return nil
		}
	}

	if err := p.PullRequestProvider.SetPromotionEnvironment(branch, env); err != nil {
		return fmt.Errorf("setting promotion for branch %s pull request to %q environment: %w", branch, env, err)
	}
	if env != "" {
		p.PromptProvider.PrintPromotionConfigured(branch, env)
	} else {
		p.PromptProvider.PrintPromotionDisabled(branch)
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
