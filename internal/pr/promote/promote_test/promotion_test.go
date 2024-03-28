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
		setExpectations func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T)
	}{
		{
			name: "master branch",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				prompt.EXPECT().PrintBranchDoesNotSupportAutoPromotion("master")

				prProvider.EnsureInstalledAndAuthenticatedFunc = func() error { return nil }
				branchProvider.GetCurrentBranchFunc = func() (string, error) { return "master", nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
				}
			},
		},
		{
			name: "branch with no PR and user opting out from creating PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return false, nil
					},
				}

				prompt.EXPECT().WhetherToCreateMissingPullRequest().Return(false, nil)
				prompt.EXPECT().PrintNotCreatingPullRequest()

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)
				}
			},
		},
		{
			name: "branch with no PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := ""
				selectedPromotionEnv := "staging"

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return false, nil
					},
					CreateInteractivelyFunc: func(branch string) error { return nil },
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return []string{}, nil
					},
					SetPromotionEnvironmentFunc: func(branch, env string) error {
						return nil
					},
				}

				prompt.EXPECT().WhetherToCreateMissingPullRequest().Return(true, nil)
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.CreateInteractivelyCalls(), 1)
					require.Equal(t, someBranch, prProvider.CreateInteractivelyCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)

					require.Len(t, prProvider.SetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.SetPromotionEnvironmentCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prProvider.SetPromotionEnvironmentCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR but no promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := ""
				selectedPromotionEnv := "staging"

				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return true, nil
					},
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return []string{}, nil
					},
					SetPromotionEnvironmentFunc: func(branch, env string) error {
						return nil
					},
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)

					require.Len(t, prProvider.SetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.SetPromotionEnvironmentCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prProvider.SetPromotionEnvironmentCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and already configured with requested promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "demo"
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().PrintPromotionAlreadyConfigured(someBranch, selectedPromotionEnv)

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return true, nil
					},
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return []string{someBranch}, nil
					},
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"
				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return true, nil
					},
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return []string{}, nil
					},
					SetPromotionEnvironmentFunc: func(branch, env string) error {
						return nil
					},
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)

					require.Len(t, prProvider.SetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.SetPromotionEnvironmentCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prProvider.SetPromotionEnvironmentCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting to disable promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"

				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[0], selectedPromotionEnv).Return(true, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[1], selectedPromotionEnv).Return(true, nil)
				prompt.EXPECT().PrintPromotionConfigured(someBranch, selectedPromotionEnv)

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error {
						return nil
					},
					ExistsFunc: func(branch string) (bool, error) {
						return true, nil
					},
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return otherBranches, nil
					},
					SetPromotionEnvironmentFunc: func(branch, env string) error {
						return nil
					},
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)

					type SetPromotionArgs = struct {
						Branch string
						Env    string
					}

					require.Len(t, prProvider.SetPromotionEnvironmentCalls(), 3)
					require.Equal(t, SetPromotionArgs{Branch: otherBranches[0]}, prProvider.SetPromotionEnvironmentCalls()[0])
					require.Equal(t, SetPromotionArgs{Branch: otherBranches[1]}, prProvider.SetPromotionEnvironmentCalls()[1])
					require.Equal(t, SetPromotionArgs{Branch: someBranch, Env: selectedPromotionEnv}, prProvider.SetPromotionEnvironmentCalls()[2])
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting out of disabling promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.MockPromptProvider) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"

				prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvNames, currentPromotionEnv).Return(selectedPromotionEnv, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[0], selectedPromotionEnv).Return(true, nil)
				prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranches[1], selectedPromotionEnv).Return(false, nil)
				prompt.EXPECT().PrintPromotionNotConfigured(someBranch, selectedPromotionEnv)

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error {
						return nil
					},
					ExistsFunc: func(branch string) (bool, error) {
						return true, nil
					},
					GetPromotionEnvironmentFunc: func(branch string) (string, error) {
						return currentPromotionEnv, nil
					},
					GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
						return otherBranches, nil
					},
					SetPromotionEnvironmentFunc: func(branch, env string) error {
						return nil
					},
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)

					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)

					require.Len(t, prProvider.GetPromotionEnvironmentCalls(), 1)
					require.Equal(t, someBranch, prProvider.GetPromotionEnvironmentCalls()[0].Branch)

					require.Len(t, prProvider.GetBranchesPromotingToEnvironmentCalls(), 1)
					require.Equal(t, selectedPromotionEnv, prProvider.GetBranchesPromotingToEnvironmentCalls()[0].Env)

					type SetPromotionArgs = struct {
						Branch string
						Env    string
					}

					require.Len(t, prProvider.SetPromotionEnvironmentCalls(), 1)
					require.Equal(t, SetPromotionArgs{Branch: otherBranches[0]}, prProvider.SetPromotionEnvironmentCalls()[0])
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
			prProvider := new(pr.PullRequestProviderMock)
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
