package rerender

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/catalog/loader"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/pkg/catalog"
)

func newLegacyPullRequest() *pr.PullRequest {
	return &pr.PullRequest{
		Number:  72789,
		Branch:  "promote-public-application-from-qa-to-production-1401301d-1454-41ac-b052-254b21054a8e",
		Labels:  []string{"environment:production", "release:public-application"},
		Body:    "\nTriggered by: john.doe -> b724eda4e96a93417e54dc8610e0f413bc395c44\n\n# Promotions\n\n# Rendering Errors: boom",
		Commits: []string{"7bf01f97bc30a80326d4fd12e502a79aee6e2e92"},
	}
}

// newSpecificReRenderer returns a re-renderer for a specific pull request, recording the pull requests fetched
// to load their catalog, which is the first step of re-rendering past selection.
func newSpecificReRenderer(loaded *[]int) ReRenderer {
	return ReRenderer{
		PullRequestProvider: &pr.PullRequestProviderMock{
			GetFunc: func(number int) (*pr.PullRequest, error) {
				pullRequest := newLegacyPullRequest()
				pullRequest.Number = number
				pullRequest.URL = "https://github.com/nestoca/catalog/pull/72789"
				return pullRequest, nil
			},
			FetchHeadFunc: func(number int) error {
				*loaded = append(*loaded, number)
				return nil
			},
		},
		CatalogLoader: newLoaderMock(),
		Out:           io.Discard,
	}
}

func newLoaderMock() *loader.LoaderMock {
	return &loader.LoaderMock{
		LoadAtFunc: func(ctx context.Context, ref string) (*catalog.Catalog, func() error, error) {
			return &catalog.Catalog{}, func() error { return nil }, nil
		},
	}
}

func TestReRenderLoadsCatalogAtParentOfFirstCommit(t *testing.T) {
	var loaded []int
	reRenderer := newSpecificReRenderer(&loaded)
	catalogLoader := newLoaderMock()
	reRenderer.CatalogLoader = catalogLoader

	_ = reRenderer.ReRender(context.Background(), "72789")
	require.Equal(t, []int{72789}, loaded)
	require.Len(t, catalogLoader.LoadAtCalls(), 1)
	require.Equal(t, "7bf01f97bc30a80326d4fd12e502a79aee6e2e92^", catalogLoader.LoadAtCalls()[0].Ref)
}

func TestReRenderPullRequestWithoutCommits(t *testing.T) {
	var loaded []int
	reRenderer := newSpecificReRenderer(&loaded)
	provider := reRenderer.PullRequestProvider.(*pr.PullRequestProviderMock)
	provider.GetFunc = func(number int) (*pr.PullRequest, error) {
		pullRequest := newLegacyPullRequest()
		pullRequest.Commits = nil
		return pullRequest, nil
	}

	require.ErrorContains(t, reRenderer.ReRender(context.Background(), "72789"), "pull request has no commits")
	require.Empty(t, loaded)
}

func TestReRenderSpecificPullRequestByNumber(t *testing.T) {
	var loaded []int
	reRenderer := newSpecificReRenderer(&loaded)

	_ = reRenderer.ReRender(context.Background(), "123")
	require.Equal(t, []int{123}, loaded)
}

func TestReRenderByURL(t *testing.T) {
	t.Run("accepts catalog pull request url", func(t *testing.T) {
		var loaded []int
		reRenderer := newSpecificReRenderer(&loaded)

		_ = reRenderer.ReRender(context.Background(), "https://github.com/NestoCa/catalog/pull/72789")
		require.Equal(t, []int{72789}, loaded)
	})

	t.Run("rejects pull request url from another repository", func(t *testing.T) {
		var loaded []int
		reRenderer := newSpecificReRenderer(&loaded)

		require.EqualError(t, reRenderer.ReRender(context.Background(), "https://github.com/nestoca/joy/pull/72789"),
			"pull request https://github.com/nestoca/joy/pull/72789 is not in catalog repository github.com/nestoca/catalog")
		require.Empty(t, loaded)
	})

	t.Run("rejects pull request url from another host", func(t *testing.T) {
		var loaded []int
		reRenderer := newSpecificReRenderer(&loaded)

		require.EqualError(t, reRenderer.ReRender(context.Background(), "https://ghe.example.com/nestoca/catalog/pull/72789"),
			"pull request https://ghe.example.com/nestoca/catalog/pull/72789 is not in catalog repository github.com/nestoca/catalog")
		require.Empty(t, loaded)
	})
}

func TestReRenderScanSkipsApprovedAndNonPromotionPullRequests(t *testing.T) {
	var loaded []int
	reRenderer := ReRenderer{
		PullRequestProvider: &pr.PullRequestProviderMock{
			ListOpenFunc: func(label string, bodyContains string) ([]pr.PullRequest, error) {
				require.Equal(t, "custom:label", label)
				require.Empty(t, bodyContains)
				approved := *newLegacyPullRequest()
				approved.Number = 1
				approved.Approved = true
				unapproved := *newLegacyPullRequest()
				unapproved.Number = 2
				preview := *newLegacyPullRequest()
				preview.Number = 3
				preview.Branch = "preview-servicing"
				return []pr.PullRequest{approved, unapproved, preview}, nil
			},
			GetFunc: func(number int) (*pr.PullRequest, error) {
				pullRequest := newLegacyPullRequest()
				pullRequest.Number = number
				return pullRequest, nil
			},
			FetchHeadFunc: func(number int) error {
				loaded = append(loaded, number)
				return nil
			},
		},
		CatalogLoader: newLoaderMock(),
		Label:         "custom:label",
		Out:           io.Discard,
	}

	err := reRenderer.ReRender(context.Background(), "")
	require.ErrorContains(t, err, "pull request #2")
	require.Equal(t, []int{2}, loaded)
}

func TestReRenderScanFiltersByLabelAndBodyInSingleQuery(t *testing.T) {
	var queries [][2]string
	reRenderer := ReRenderer{
		PullRequestProvider: &pr.PullRequestProviderMock{
			ListOpenFunc: func(label string, bodyContains string) ([]pr.PullRequest, error) {
				queries = append(queries, [2]string{label, bodyContains})
				return nil, nil
			},
		},
		Label:        promote.RenderingFailedLabel,
		BodyContains: "No such device or address",
		Out:          io.Discard,
	}

	require.NoError(t, reRenderer.ReRender(context.Background(), ""))
	require.Equal(t, [][2]string{{promote.RenderingFailedLabel, "No such device or address"}}, queries)
}
