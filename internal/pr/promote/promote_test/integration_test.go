package promote_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/nestoca/joy/internal/github"

	"github.com/go-test/deep"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/nestoca/joy/internal/testutils"
)

var (
	promotableEnvs          = []string{"staging", "demo"}
	commonLabels            = []string{"label1", "label2", "label3"}
	possiblePromotionLabels = []string{"promote:staging", "promote:demo"}
)

func TestPromotions(t *testing.T) {
	gitRepo := testutils.CloneToTempDir(t, "joy-pr-promote-test")

	t.Run("testPromotePRs", func(t *testing.T) {
		testPromotePRs(t, gitRepo)
	})

	t.Run("testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo", func(t *testing.T) {
		testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo(t, gitRepo)
	})

	t.Run("testGetCurrentBranch", func(t *testing.T) {
		testGetCurrentBranch(t, gitRepo)
	})
}

func testPromotePRs(t *testing.T, dir string) {
	testCases := []struct {
		name                               string
		branch                             string
		oldEnv                             string
		newEnv                             string
		oldLabel                           string
		newLabel                           string
		otherBranchesAutoPromotingToNewEnv []string
	}{
		{
			name:     "Change promotion of existing PR from none to staging",
			branch:   "branch-with-pr",
			oldEnv:   "",
			newEnv:   "staging",
			oldLabel: "",
			newLabel: "promote:staging",
			otherBranchesAutoPromotingToNewEnv: []string{
				"branch-with-pr-promoting-to-staging",
			},
		},
		{
			name:     "Change promotion of existing PR from staging to demo",
			branch:   "branch-with-pr-promoting-to-staging",
			oldEnv:   "staging",
			newEnv:   "demo",
			oldLabel: "promote:staging",
			newLabel: "promote:demo",
		},
		{
			name:     "Change promotion of existing PR from staging to none",
			branch:   "branch-with-pr-promoting-to-staging",
			oldEnv:   "staging",
			newEnv:   "",
			oldLabel: "promote:staging",
			newLabel: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Preparation and clean-up
			require.NoError(t, git.Checkout(dir, tc.branch))
			setExclusivePromotionLabel(t, dir, tc.branch, tc.oldLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				setExclusivePromotionLabel(t, dir, otherBranch, tc.newLabel)
			}
			defer setExclusivePromotionLabel(t, dir, tc.branch, tc.oldLabel)

			prompt := &promote.PromptProviderMock{
				WhichEnvironmentToPromoteToFunc: func(environments []string, preSelectedEnv string) (string, error) {
					return tc.newEnv, nil
				},
				ConfirmDisablingPromotionOnOtherPullRequestFunc: func(branch, env string) (bool, error) {
					return true, nil
				},
			}
			defer func() {
				require.Len(t, prompt.WhichEnvironmentToPromoteToCalls(), 1)
				require.Equal(t, prompt.WhichEnvironmentToPromoteToCalls()[0].Environments, promotableEnvs)
				require.Equal(t, prompt.WhichEnvironmentToPromoteToCalls()[0].PreSelectedEnv, tc.oldEnv)

				require.Len(t, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls(), len(tc.otherBranchesAutoPromotingToNewEnv))
				for _, branch := range tc.otherBranchesAutoPromotingToNewEnv {
					require.Equal(t, branch, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Branch)
					require.Equal(t, tc.newEnv, prompt.ConfirmDisablingPromotionOnOtherPullRequestCalls()[0].Env, tc.newEnv)
				}

				if tc.newEnv != "" {
					require.Len(t, prompt.PrintPromotionConfiguredCalls(), 1)
					require.Equal(t, tc.branch, prompt.PrintPromotionConfiguredCalls()[0].Branch)
					require.Equal(t, tc.newEnv, prompt.PrintPromotionConfiguredCalls()[0].Env)
					require.Len(t, prompt.PrintPromotionDisabledCalls(), 0)
				} else {
					require.Len(t, prompt.PrintPromotionDisabledCalls(), 1)
					require.Equal(t, tc.branch, prompt.PrintPromotionDisabledCalls()[0].Branch)
				}
			}()

			// Perform test
			promotion := promote.NewPromotion(promote.NewGitBranchProvider(dir), github.NewPullRequestProvider(dir), prompt)
			err := promotion.Promote(newEnvironments())

			// Check results
			require.NoError(t, err)
			requireLabel(t, dir, tc.branch, tc.newLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				requireLabel(t, dir, otherBranch, "")
			}
		})
	}
}

func testSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo(t *testing.T, dir string) {
	branch := "branch-with-pr"
	guid, err := uuid.NewRandom()
	require.NoError(t, err)
	expectedEnv := guid.String()
	require.NoError(t, git.Checkout(dir, branch))
	prProvider := github.NewPullRequestProvider(dir)

	err = prProvider.SetPromotionEnvironment(branch, expectedEnv)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prProvider.SetPromotionEnvironment(branch, "")
		deletePromotionLabelFromRepo(t, dir, expectedEnv)
	})

	actualEnv, err := prProvider.GetPromotionEnvironment(branch)
	require.NoError(t, err)
	require.Equal(t, expectedEnv, actualEnv)
}

func testGetCurrentBranch(t *testing.T, dir string) {
	// Prepare
	expectedBranch := "branch-with-pr"
	branchProvider := promote.NewGitBranchProvider(dir)
	require.NoError(t, git.Checkout(dir, expectedBranch))

	// Perform test
	actualBranch, err := branchProvider.GetCurrentBranch()

	// Require
	require.NoError(t, err, "getting current branch")
	require.Equal(t, expectedBranch, actualBranch)
}

func addLabel(t *testing.T, dir, branch string, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", branch, "--add-label", strings.Join(labels, ","))
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "adding labels %v", labels)
}

func removeLabel(t *testing.T, dir, branch string, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", branch, "--remove-label", strings.Join(labels, ","))
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "remove labels %v", labels)
}

func deletePromotionLabelFromRepo(t *testing.T, dir, env string) {
	t.Helper()
	label := fmt.Sprintf("promote:%s", env)
	cmd := exec.Command("gh", "label", "delete", label, "--yes")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "delete label %s from repo", label)
}

type pullRequest struct {
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func requireLabel(t *testing.T, dir, branch, expectedLabel string) {
	t.Helper()

	cmd := exec.Command("gh", "pr", "list", "--head", branch, "--json", "labels")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	require.NoError(t, err, "listing labels")

	// Always include common labels
	expectedLabels := commonLabels
	if expectedLabel != "" {
		expectedLabels = append(expectedLabels, expectedLabel)
	}

	// Unmarshal JSON
	var prs []pullRequest
	if err := json.Unmarshal(out.Bytes(), &prs); err != nil {
		t.Error(err)
	}

	// Extract labels
	var actualLabels []string
	for _, pr := range prs {
		for _, label := range pr.Labels {
			actualLabels = append(actualLabels, label.Name)
		}
	}

	// Ensure we got exactly expected labels
	sort.Strings(expectedLabels)
	sort.Strings(actualLabels)
	if diff := deep.Equal(expectedLabels, actualLabels); diff != nil {
		t.Error(diff)
	}
}

func setExclusivePromotionLabel(t *testing.T, dir, branch, label string) {
	t.Helper()

	for _, possibleLabel := range possiblePromotionLabels {
		if possibleLabel == label {
			addLabel(t, dir, branch, possibleLabel)
		} else {
			removeLabel(t, dir, branch, possibleLabel)
		}
	}
}
