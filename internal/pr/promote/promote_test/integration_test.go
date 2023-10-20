package promote_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-test/deep"
	"github.com/google/uuid"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

var promotableEnvs = []string{"staging", "demo"}
var commonLabels = []string{"label1", "label2", "label3"}
var possiblePromotionLabels = []string{"promote:staging", "promote:demo"}

func TestPromotePRs(t *testing.T) {
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
			// Create mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			prompt := promote.NewMockPromptProvider(ctrl)

			// Preparation and clean-up
			assert.NoError(t, git.Checkout(".", tc.branch))
			setExclusivePromotionLabel(t, tc.branch, tc.oldLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				setExclusivePromotionLabel(t, otherBranch, tc.newLabel)
			}
			defer setExclusivePromotionLabel(t, tc.branch, tc.oldLabel)

			// Set expectations
			prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvs, tc.oldEnv).Return(tc.newEnv, nil)
			if len(tc.otherBranchesAutoPromotingToNewEnv) > 0 {
				for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
					prompt.EXPECT().ConfirmDisablingPromotionOnOtherPullRequest(otherBranch, tc.newEnv).Return(true, nil)
				}
			}
			if tc.newEnv != "" {
				prompt.EXPECT().PrintPromotionConfigured(tc.branch, tc.newEnv)
			} else {
				prompt.EXPECT().PrintPromotionDisabled(tc.branch)
			}

			// Perform test
			promotion := promote.NewPromotion(&promote.GitBranchProvider{}, &promote.GitHubPullRequestProvider{}, prompt)
			err := promotion.Promote(newEnvironments())

			// Check results
			assert.NoError(t, err)
			assertLabel(t, tc.branch, tc.newLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				assertLabel(t, otherBranch, "")
			}
		})
	}
}

func TestSettingAutoPromotionEnvUsingLabelNotAlreadyExistingInRepo(t *testing.T) {
	branch := "branch-with-pr"
	guid, err := uuid.NewRandom()
	assert.NoError(t, err)
	expectedEnv := guid.String()
	checkOut(t, branch)
	prProvider := promote.GitHubPullRequestProvider{}

	err = prProvider.SetPromotionEnvironment(branch, expectedEnv)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = prProvider.SetPromotionEnvironment(branch, "")
		deletePromotionLabelFromRepo(t, expectedEnv)
	})

	actualEnv, err := prProvider.GetPromotionEnvironment(branch)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, actualEnv)
}

func addLabel(t *testing.T, branch string, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", branch, "--add-label", strings.Join(labels, ","))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "adding labels %v", labels)
}

func removeLabel(t *testing.T, branch string, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", branch, "--remove-label", strings.Join(labels, ","))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "remove labels %v", labels)
}

func deletePromotionLabelFromRepo(t *testing.T, env string) {
	t.Helper()
	label := fmt.Sprintf("promote:%s", env)
	cmd := exec.Command("gh", "label", "delete", label, "--yes")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "delete label %s from repo", label)
}

type pullRequest struct {
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func assertLabel(t *testing.T, branch, expectedLabel string) {
	cmd := exec.Command("gh", "pr", "list", "--head", branch, "--json", "labels")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	assert.NoError(t, err, "listing labels")

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

func setExclusivePromotionLabel(t *testing.T, branch, label string) {
	for _, possibleLabel := range possiblePromotionLabels {
		if possibleLabel == label {
			addLabel(t, branch, possibleLabel)
		} else {
			removeLabel(t, branch, possibleLabel)
		}
	}
}
