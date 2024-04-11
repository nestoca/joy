package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/go-test/deep"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/pkg/catalog"
)

var (
	commonLabels            = []string{"label1", "label2", "label3"}
	possiblePromotionLabels = []string{"promote:staging", "promote:demo"}
)

func TestPromotePRs(t *testing.T) {
	projectDir := testutils.CloneToTempDir(t, "joy-pr-promote-test")

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
			require.NoError(t, git.Checkout(projectDir, tc.branch))
			setExclusivePromotionLabel(t, projectDir, tc.branch, tc.oldLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				setExclusivePromotionLabel(t, projectDir, otherBranch, tc.newLabel)
			}
			defer setExclusivePromotionLabel(t, projectDir, tc.branch, tc.oldLabel)

			args := []string{"--no-prompt"}
			if tc.newEnv == "" {
				args = append(args, "--disable")
			} else {
				args = append(args, "--target", tc.newEnv)
			}

			_, err := executePRPromoteCommand(t, projectDir, NewPRPromoteCmd(), args...)
			require.NoError(t, err)
			requireLabel(t, projectDir, tc.branch, tc.newLabel)
			for _, otherBranch := range tc.otherBranchesAutoPromotingToNewEnv {
				requireLabel(t, projectDir, otherBranch, "")
			}
		})
	}
}

func executePRPromoteCommand(t *testing.T, projectDir string, cmd *cobra.Command, args ...string) (string, error) {
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
	cat := builder.Build()
	cfg := &config.Config{}

	ctx := config.ToContext(catalog.ToContext(context.Background(), cat), cfg)

	var buffer bytes.Buffer
	cmd.SetOut(&buffer)
	cmd.SetArgs(args)
	cmd.SetContext(ctx)

	wd, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(projectDir))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	err = cmd.Execute()
	return stripansi.Strip(buffer.String()), err
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

	expectedLabels := commonLabels
	if expectedLabel != "" {
		expectedLabels = append(expectedLabels, expectedLabel)
	}

	var prs []pullRequest
	if err := json.Unmarshal(out.Bytes(), &prs); err != nil {
		t.Error(err)
	}

	var actualLabels []string
	for _, pr := range prs {
		for _, label := range pr.Labels {
			actualLabels = append(actualLabels, label.Name)
		}
	}

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
