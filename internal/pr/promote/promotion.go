package promote

import (
	"fmt"

	"github.com/nestoca/joy/internal/git/pr/github"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
)

type Promotion struct {
	// branchProvider is the provider for managing git branches
	branchProvider BranchProvider

	// pullRequestProvider is the provider of pull requests
	pullRequestProvider pr.PullRequestProvider

	// promptProvider is the prompt to use for user interaction
	promptProvider PromptProvider
}

func NewPromotion(branchProvider BranchProvider, pullRequestProvider pr.PullRequestProvider, prompt PromptProvider) *Promotion {
	return &Promotion{
		branchProvider:      branchProvider,
		pullRequestProvider: pullRequestProvider,
		promptProvider:      prompt,
	}
}

func NewDefaultPromotion(dir string) *Promotion {
	return NewPromotion(
		NewGitBranchProvider(dir),
		github.NewPullRequestProvider(dir),
		&InteractivePromptProvider{},
	)
}

// Promote prompts user to create a pull request for current branch and to select environment to auto-promote builds
// of pull request to, and then configures the pull request accordingly.
func (p *Promotion) Promote(environments []*v1alpha1.Environment) error {
	if err := p.pullRequestProvider.EnsureInstalledAndAuthenticated(); err != nil {
		return nil
	}

	branch, err := p.branchProvider.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if branch == "master" || branch == "main" {
		p.promptProvider.PrintBranchDoesNotSupportAutoPromotion(branch)
		return nil
	}

	exists, err := p.pullRequestProvider.Exists(branch)
	if err != nil {
		return fmt.Errorf("checking if pull request exists for branch %s: %w", branch, err)
	}

	if !exists {
		shouldCreate, err := p.promptProvider.WhetherToCreateMissingPullRequest()
		if err != nil {
			return fmt.Errorf("prompting user to create pull request: %w", err)
		}
		if !shouldCreate {
			p.promptProvider.PrintNotCreatingPullRequest()
			return nil
		}

		err = p.pullRequestProvider.CreateInteractively(branch)
		if err != nil {
			return fmt.Errorf("creating pull request: %w", err)
		}
	}

	env, err := p.pullRequestProvider.GetPromotionEnvironment(branch)
	if err != nil {
		return fmt.Errorf("getting promotion environment for branch %s pull request: %w", branch, err)
	}

	fmt.Println("using promotion environment:", env)

	promotableEnvironmentNames := getPromotableEnvironmentNames(environments)
	env, err = p.promptProvider.WhichEnvironmentToPromoteTo(promotableEnvironmentNames, env)
	if err != nil {
		return fmt.Errorf("prompting user to select promotion environment: %w", err)
	}

	// Disable auto-promotion on other pull requests promoting to same environment
	branchesPromotingToEnv, err := p.pullRequestProvider.GetBranchesPromotingToEnvironment(env)
	if err != nil {
		return fmt.Errorf("getting branches configured for auto-promotion to %q environment: %w", env, err)
	}
	for _, branchPromotingToEnv := range branchesPromotingToEnv {
		if branchPromotingToEnv == branch {
			p.promptProvider.PrintPromotionAlreadyConfigured(branch, env)
			return nil
		}
		shouldDisable, err := p.promptProvider.ConfirmDisablingPromotionOnOtherPullRequest(branchPromotingToEnv, env)
		if err != nil {
			return fmt.Errorf("prompting user to confirm disabling promotion on other pull request: %w", err)
		}
		if shouldDisable {
			if err := p.pullRequestProvider.SetPromotionEnvironment(branchPromotingToEnv, ""); err != nil {
				return fmt.Errorf("disabling promotion for branch %s pull request: %w", branchPromotingToEnv, err)
			}
		} else {
			p.promptProvider.PrintPromotionNotConfigured(branch, env)
			return nil
		}
	}

	if err := p.pullRequestProvider.SetPromotionEnvironment(branch, env); err != nil {
		return fmt.Errorf("setting promotion for branch %s pull request to %q environment: %w", branch, env, err)
	}
	if env != "" {
		p.promptProvider.PrintPromotionConfigured(branch, env)
	} else {
		p.promptProvider.PrintPromotionDisabled(branch)
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
