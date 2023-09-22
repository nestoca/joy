package promote_test

import (
	"bytes"
	"github.com/go-test/deep"
	"github.com/nestoca/joy/internal/pr/promote"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/slices"
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
		name     string
		branch   string
		oldEnv   string
		newEnv   string
		oldLabel string
		newLabel string
	}{
		{
			name:     "Change promotion of existing PR from none to staging",
			branch:   "branch-with-pr",
			oldEnv:   "",
			newEnv:   "staging",
			oldLabel: "",
			newLabel: "promote:staging",
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
			checkOut(t, tc.branch)
			setExclusivePromotionLabel(t, tc.oldLabel)
			defer setExclusivePromotionLabel(t, tc.oldLabel)

			// Set expectations
			prompt.EXPECT().WhichEnvironmentToPromoteTo(promotableEnvs, tc.oldEnv).Return(tc.newEnv, nil)
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
			assertLabel(t, tc.newLabel)
		})
	}
}

func addLabels(t *testing.T, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", "--add-label", strings.Join(labels, ","))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "adding labels %v", labels)
}

func removeLabels(t *testing.T, labels ...string) {
	t.Helper()
	cmd := exec.Command("gh", "pr", "edit", "--remove-label", strings.Join(labels, ","))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err, "remove labels %v", labels)
}

func assertLabel(t *testing.T, expectedLabel string) {
	cmd := exec.Command("gh", "pr", "view", "--json", "labels", "-q", ".labels[].name")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	assert.NoError(t, err, "listing labels")

	// Always include common labels
	expectedLabels := commonLabels
	if expectedLabel != "" {
		expectedLabels = append(expectedLabels, expectedLabel)
	}

	// Ensure we got exactly expected labels
	actualLabels := strings.Split(strings.TrimSpace(out.String()), "\n")
	sort.Strings(expectedLabels)
	sort.Strings(actualLabels)
	if diff := deep.Equal(expectedLabels, actualLabels); diff != nil {
		t.Error(diff)
	}
}

func setExclusivePromotionLabel(t *testing.T, labels ...string) {
	for _, label := range possiblePromotionLabels {
		if slices.Contains(labels, label) {
			addLabels(t, label)
		} else {
			removeLabels(t, label)
		}
	}
}
