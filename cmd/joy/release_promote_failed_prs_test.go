package main

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestSelectFailedPullRequestsSkipsApprovedAndNonPromotionPullRequests(t *testing.T) {
	provider := &pr.PullRequestProviderMock{
		ListFunc: func(search string, limit int) ([]pr.PullRequest, error) {
			require.Equal(t, `label:"rendering:failed"`, search)
			return []pr.PullRequest{
				{Number: 1, Branch: "promote-a-from-qa-to-production-uuid"},
				{Number: 2, Branch: "promote-b-from-qa-to-production-uuid", Approved: true},
				{Number: 3, Branch: "feature-branch"},
			}, nil
		},
	}

	pullRequests, err := selectFailedPullRequests(provider, `label:"rendering:failed"`)
	require.NoError(t, err)
	require.Equal(t, []pr.PullRequest{{Number: 1, Branch: "promote-a-from-qa-to-production-uuid"}}, pullRequests)
}

func TestPullRequestBaseRef(t *testing.T) {
	cases := []struct {
		Name          string
		Commits       []string
		Expected      string
		ExpectedError string
	}{
		{
			Name:     "parent of first commit",
			Commits:  []string{"first", "second"},
			Expected: "first^",
		},
		{
			Name:          "no commits",
			ExpectedError: "pull request has no commits",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ref, err := pullRequestBaseRef(pr.PullRequest{Commits: tc.Commits})
			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.Expected, ref)
		})
	}
}

func TestUpdateFailedPullRequestsLoadsCatalogAtEachPullRequestBaseThenCatalogRef(t *testing.T) {
	provider := &pr.PullRequestProviderMock{
		ListFunc: func(string, int) ([]pr.PullRequest, error) {
			return []pr.PullRequest{
				{
					Number:  1,
					Branch:  "promote-a-from-qa-to-production-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					Labels:  []string{"environment:production", "release:a"},
					Commits: []string{"a1", "a2"},
				},
				{
					Number:  2,
					Branch:  "promote-b-from-qa-to-production-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					Labels:  []string{"environment:production", "release:b"},
					Commits: []string{"b1"},
				},
			}, nil
		},
	}

	var loadedRefs []string
	params := PromoteParams{
		PullRequest: provider,
		LoadCatalogAt: func(_ context.Context, cfg *config.Config, ref string, _ io.Writer) (*config.Config, *catalog.Catalog, error) {
			loadedRefs = append(loadedRefs, ref)
			return cfg, &catalog.Catalog{}, nil
		},
	}

	ctx := config.ToContext(context.Background(), &config.Config{})
	ctx = config.ToFlagsContext(ctx, &config.GlobalFlags{CatalogRef: "release/v1"})
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := updateFailedPullRequests(cmd, params, `label:"rendering:failed"`, true)

	require.ErrorContains(t, err, `environment "qa" not found`)
	require.Equal(t, []string{"a1^", "b1^", "release/v1"}, loadedRefs)
}
