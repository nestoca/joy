package rerender

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/release/promote"
)

func TestResolveMetadataFromBody(t *testing.T) {
	pullRequest := newLegacyPullRequest()
	pullRequest.Body = promote.AppendPromotionMetadata(pullRequest.Body, promote.PromotionMetadata{Source: "staging", Target: "demo", Releases: []string{"other"}})

	metadata, err := ResolveMetadata(pullRequest)
	require.NoError(t, err)
	require.Equal(t, &promote.PromotionMetadata{Source: "staging", Target: "demo", Releases: []string{"other"}}, metadata)
}

func TestResolveMetadataFromLegacyPullRequest(t *testing.T) {
	metadata, err := ResolveMetadata(newLegacyPullRequest())
	require.NoError(t, err)
	require.Equal(t, &promote.PromotionMetadata{
		Source:    "qa",
		Target:    "production",
		Releases:  []string{"public-application"},
		Variables: map[string]string{"triggeredBy": "john.doe -> b724eda4e96a93417e54dc8610e0f413bc395c44"},
		Reviewers: []string{"john.doe"},
	}, metadata)
}

func TestResolveMetadataFromLegacyPullRequestWithDashedEnvironments(t *testing.T) {
	pullRequest := newLegacyPullRequest()
	pullRequest.Branch = "promote-2-releases-from-mktg-dev-to-mktg-prod-1401301d-1454-41ac-b052-254b21054a8e"
	pullRequest.Labels = []string{"environment:mktg-prod", "release:a", "release:b"}

	metadata, err := ResolveMetadata(pullRequest)
	require.NoError(t, err)
	require.Equal(t, "mktg-dev", metadata.Source)
	require.Equal(t, "mktg-prod", metadata.Target)
	require.Equal(t, []string{"a", "b"}, metadata.Releases)
}

func TestResolveMetadataWithoutLabels(t *testing.T) {
	pullRequest := newLegacyPullRequest()
	pullRequest.Labels = nil

	_, err := ResolveMetadata(pullRequest)
	require.ErrorContains(t, err, "no promotion metadata")
}
