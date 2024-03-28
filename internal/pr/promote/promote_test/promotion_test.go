package promote_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/pr/promote"
)

func newEnvironment(name string, promotable bool) *v1alpha1.Environment {
	return &v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
			Name: name,
		},
		Spec: v1alpha1.EnvironmentSpec{
			Promotion: v1alpha1.Promotion{
				FromPullRequests: promotable,
			},
		},
	}
}

func newEnvironments() []*v1alpha1.Environment {
	return []*v1alpha1.Environment{
		newEnvironment("staging", true),
		newEnvironment("qa", false),
		newEnvironment("production", false),
		newEnvironment("demo", true),
	}
}

func TestPromotion(t *testing.T) {
	cases := []struct {
		name            string
		setExpectations func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T)
	}{
		{
			name: "master branch",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prompt.EXPECT().PrintBranchDoesNotSupportAutoPromotion("master")

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return "master", nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with no PR and user opting out from creating PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(false, nil)
				prompt.EXPECT().WhetherToCreateMissingPullRequest().Return(false, nil)
				prompt.EXPECT().PrintNotCreatingPullRequest()

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with no PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := ""
				selectedPromotionEnv := "staging"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(false, nil)
				prompt.EXPECT().WhetherToCreateMissingPullRequest().Return(true, nil)
				prProvider.EXPECT().CreateInteractively(someBranch).Return(nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return([]string{}, nil)
				prProvider.EXPECT().SetPromotionEnvironment(someBranch, selectedPromotionEnv).Return(nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with existing PR but no promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := ""
				selectedPromotionEnv := "staging"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(true, nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return([]string{}, nil)
				prProvider.EXPECT().SetPromotionEnvironment(someBranch, selectedPromotionEnv).Return(nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with existing PR and already configured with requested promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "demo"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(true, nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return([]string{someBranch}, nil)
				prompt.EXPECT().PrintPromotionAlreadyConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(true, nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return([]string{}, nil)
				prProvider.EXPECT().SetPromotionEnvironment(someBranch, selectedPromotionEnv).Return(nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting to disable promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(true, nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return(otherBranches, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[0], selectedPromotionEnv).Return(true, nil)
				prProvider.EXPECT().SetPromotionEnvironment(otherBranches[0], "").Return(nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[1], selectedPromotionEnv).Return(true, nil)
				prProvider.EXPECT().SetPromotionEnvironment(otherBranches[1], "").Return(nil)
				prProvider.EXPECT().SetPromotionEnvironment(someBranch, selectedPromotionEnv).Return(nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting out of disabling promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.MockPullRequestProvider, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"
				prProvider.EXPECT().EnsureInstalledAndAuthenticated().Return(nil)
				prProvider.EXPECT().Exists(someBranch).Return(true, nil)
				prProvider.EXPECT().GetPromotionEnvironment(someBranch).Return(currentPromotionEnv, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prProvider.EXPECT().GetBranchesPromotingToEnvironment(selectedPromotionEnv).Return(otherBranches, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[0], selectedPromotionEnv).Return(true, nil)
				prProvider.EXPECT().SetPromotionEnvironment(otherBranches[0], "").Return(nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[1], selectedPromotionEnv).Return(false, nil)
				prompt.EXPECT().PrintPromotionNotConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			branchProvider := new(promote.BranchProviderMock)
			prProvider := pr.NewMockPullRequestProvider(ctrl)
			prompt := promote.NewMockPromptProvider(ctrl)

			// Set case-specific expectations
			defer c.setExpectations(branchProvider, prProvider, prompt)(t)

			// Perform test
			promotion := promote.NewPromotion(branchProvider, prProvider, prompt)
			environments := newEnvironments()
			err := promotion.Promote(environments)
			assert.NoError(t, err)
		})
	}
}
