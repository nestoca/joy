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

func TestSelectFailedPullRequestsSkipsOpenApprovedAndNonPromotionPullRequests(t *testing.T) {
	provider := &pr.PullRequestProviderMock{
		ListFunc: func(search string, limit int) ([]pr.PullRequest, error) {
			require.Equal(t, `label:"rendering:failed"`, search)
			return []pr.PullRequest{
				{Number: 1, Branch: "promote-a-from-qa-to-production-uuid", Open: true},
				{Number: 2, Branch: "promote-b-from-qa-to-production-uuid", Open: true, Approved: true},
				{Number: 3, Branch: "feature-branch", Open: true},
				{Number: 4, Branch: "promote-c-from-qa-to-production-uuid", Approved: true},
				{Number: 5, Branch: "promote-d-from-qa-to-production-uuid"},
			}, nil
		},
	}

	pullRequests, err := selectFailedPullRequests(provider, `label:"rendering:failed"`)
	require.NoError(t, err)
	require.Equal(t, []pr.PullRequest{
		{Number: 1, Branch: "promote-a-from-qa-to-production-uuid", Open: true},
		{Number: 4, Branch: "promote-c-from-qa-to-production-uuid", Approved: true},
		{Number: 5, Branch: "promote-d-from-qa-to-production-uuid"},
	}, pullRequests)
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
	require.Len(t, provider.FetchHeadCalls(), 2)
	require.Equal(t, 1, provider.FetchHeadCalls()[0].Number)
	require.Equal(t, 2, provider.FetchHeadCalls()[1].Number)
}

func TestReleasePromoteRejectsFlagsIgnoredByUpdateFailedPullRequests(t *testing.T) {
	cases := []struct {
		Name          string
		Args          []string
		ExpectedError string
	}{
		{
			Name:          "reviewers",
			Args:          []string{"--update-failed-prs", "--reviewers", "john.doe"},
			ExpectedError: "flag --reviewers cannot be used with --update-failed-prs",
		},
		{
			Name:          "template var",
			Args:          []string{"--update-failed-prs", "--template-var", "key=value"},
			ExpectedError: "flag --template-var cannot be used with --update-failed-prs",
		},
		{
			Name:          "omit",
			Args:          []string{"--update-failed-prs", "--omit", "release"},
			ExpectedError: "flag --omit cannot be used with --update-failed-prs",
		},
		{
			Name:          "keep prerelease",
			Args:          []string{"--update-failed-prs", "--keep-prerelease"},
			ExpectedError: "flag --keep-prerelease cannot be used with --update-failed-prs",
		},
		{
			Name:          "auto merge",
			Args:          []string{"--update-failed-prs", "--auto-merge"},
			ExpectedError: "flag --auto-merge cannot be used with --update-failed-prs",
		},
		{
			Name:          "search without update failed prs",
			Args:          []string{"--update-failed-prs-search", "label:foo", "--source", "staging", "--target", "prod", "foo"},
			ExpectedError: "flag --update-failed-prs-search requires --update-failed-prs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			cmd := NewReleasePromoteCmd(PromoteParams{PreRunConfigs: make(PreRunConfigs)})
			cmd.SetArgs(tc.Args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SilenceUsage = true

			require.EqualError(t, cmd.Execute(), tc.ExpectedError)
		})
	}
}
