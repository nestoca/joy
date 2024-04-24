package promote_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/nestoca/joy/pkg/catalog"
)

func newEnvironments(t *testing.T) []*v1alpha1.Environment {
	promotable := func(e *v1alpha1.Environment) {
		e.Spec.Promotion.FromPullRequests = true
	}
	notPromotable := func(e *v1alpha1.Environment) {
		e.Spec.Promotion.FromPullRequests = false
	}

	builder := catalog.NewBuilder(t)
	builder.AddEnvironment("staging", promotable)
	builder.AddEnvironment("qa", notPromotable)
	builder.AddEnvironment("production", notPromotable)
	builder.AddEnvironment("demo", promotable)
	return builder.Build().Environments
}

func TestPromotion(t *testing.T) {
	cases := []struct {
		name            string
		setExpectations func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T)
	}{
		{
			name: "master branch",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				prProvider.EnsureInstalledAndAuthenticatedFunc = func() error { return nil }
				branchProvider.GetCurrentBranchFunc = func() (string, error) { return "master", nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)

					require.Len(t, prompt.PrintBranchDoesNotSupportAutoPromotionCalls(), 1)
					require.Equal(t, "master", prompt.PrintBranchDoesNotSupportAutoPromotionCalls()[0].Branch)
				}
			},
		},
		{
			name: "branch with no PR and user opting out from creating PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"

				*prProvider = pr.PullRequestProviderMock{
					EnsureInstalledAndAuthenticatedFunc: func() error { return nil },
					ExistsFunc: func(branch string) (bool, error) {
						return false, nil
					},
				}

				prompt.WhetherToCreateMissingPullRequestFunc = func() (bool, error) {
					return false, nil
				}

				branchProvider.GetCurrentBranchFunc = func() (string, error) { return someBranch, nil }

				return func(t *testing.T) {
					require.Len(t, branchProvider.GetCurrentBranchCalls(), 1)
					require.Len(t, prProvider.EnsureInstalledAndAuthenticatedCalls(), 1)
					require.Len(t, prProvider.ExistsCalls(), 1)
					require.Equal(t, someBranch, prProvider.ExistsCalls()[0].Branch)
					require.Len(t, prompt.WhetherToCreateMissingPullRequestCalls(), 1)
					require.Len(t, prompt.PrintNotCreatingPullRequestCalls(), 1)
				}
			},
		},
		{
			name: "branch with no PR",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
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

				*prompt = promote.PromptProviderMock{
					WhetherToCreateMissingPullRequestFunc: func() (bool, error) {
						return true, nil
					},
					WhichEnvironmentToPromoteToFunc: func(environments []string, preSelectedEnv string) (string, error) {
						return selectedPromotionEnv, nil
					},
				}

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

					require.Len(t, prompt.WhetherToCreateMissingPullRequestCalls(), 1)
					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)

					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, currentPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)
				}
			},
		},
		{
			name: "branch with existing PR but no promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := ""
				selectedPromotionEnv := "staging"

				prompt.WhichEnvironmentToPromoteToFunc = func(environments []string, preSelectedEnv string) (string, error) {
					return selectedPromotionEnv, nil
				}

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

					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, currentPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)

					require.Len(t, prompt.PrintPromotionConfiguredCalls(), 1)
					require.Equal(t, someBranch, prompt.PrintPromotionConfiguredCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.PrintPromotionConfiguredCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and already configured with requested promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "demo"

				prompt.WhichEnvironmentToPromoteToFunc = func(environments []string, preSelectedEnv string) (string, error) {
					return selectedPromotionEnv, nil
				}

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

					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, selectedPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)

					require.Len(t, prompt.PrintPromotionAlreadyConfiguredCalls(), 1)
					require.Equal(t, someBranch, prompt.PrintPromotionAlreadyConfiguredCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.PrintPromotionAlreadyConfiguredCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"

				prompt.WhichEnvironmentToPromoteToFunc = func(environments []string, preSelectedEnv string) (string, error) {
					return selectedPromotionEnv, nil
				}

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

					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, currentPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)

					require.Len(t, prompt.PrintPromotionConfiguredCalls(), 1)
					require.Equal(t, someBranch, prompt.PrintPromotionConfiguredCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.PrintPromotionConfiguredCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting to disable promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"

				*prompt = promote.PromptProviderMock{
					WhichEnvironmentToPromoteToFunc: func(environments []string, preSelectedEnv string) (string, error) {
						return selectedPromotionEnv, nil
					},
					ConfirmDisablingPromotionOnOtherPullRequestFunc: func(branch, env string) (bool, error) {
						return true, nil
					},
				}

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

					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, currentPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)

					require.Len(t, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls(), 2)
					require.Equal(t, otherBranches[0], prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Env)
					require.Equal(t, otherBranches[1], prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[1].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[1].Env)

					require.Len(t, prompt.PrintPromotionConfiguredCalls(), 1)
					require.Equal(t, someBranch, prompt.PrintPromotionConfiguredCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.PrintPromotionConfiguredCalls()[0].Env)
				}
			},
		},
		{
			name: "branch with existing PR and promotion env with user opting out of disabling promotion on other PRs",
			setExpectations: func(branchProvider *promote.BranchProviderMock, prProvider *pr.PullRequestProviderMock, prompt *promote.PromptProviderMock) func(t *testing.T) {
				someBranch := "some-branch"
				otherBranches := []string{"other-branch1", "other-branch2"}
				promotableEnvNames := []string{"staging", "demo"}
				currentPromotionEnv := "demo"
				selectedPromotionEnv := "staging"

				*prompt = promote.PromptProviderMock{
					WhichEnvironmentToPromoteToFunc: func(environments []string, preSelectedEnv string) (string, error) {
						return selectedPromotionEnv, nil
					},
					ConfirmDisablingPromotionOnOtherPullRequestFunc: func() func(branch, env string) (bool, error) {
						var count int
						return func(branch, env string) (bool, error) {
							count++
							if count == 1 {
								return true, nil
							}
							return false, nil
						}
					}(),
				}

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

					require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
					require.Equal(t, promotableEnvNames, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments)
					require.Equal(t, currentPromotionEnv, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv)

					require.Len(t, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls(), 2)
					require.Equal(t, otherBranches[0], prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Env)
					require.Equal(t, otherBranches[1], prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[1].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[1].Env)

					require.Len(t, prompt.PrintPromotionNotConfiguredCalls(), 1)
					require.Equal(t, someBranch, prompt.PrintPromotionNotConfiguredCalls()[0].Branch)
					require.Equal(t, selectedPromotionEnv, prompt.PrintPromotionNotConfiguredCalls()[0].Env)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			branchProvider := new(promote.BranchProviderMock)
			prProvider := new(pr.PullRequestProviderMock)
			prompt := new(promote.PromptProviderMock)

			// Set case-specific expectations
			defer c.setExpectations(branchProvider, prProvider, prompt)(t)

			// Perform test
			promotion := promote.NewPromotion(branchProvider, prProvider, prompt)
			environments := newEnvironments(t)
			err := promotion.Promote(promote.Params{
				Environments: environments,
			})
			assert.NoError(t, err)
		})
	}
}
