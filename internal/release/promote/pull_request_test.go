package promote

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/git/pr"
)

func newPromotionPullRequest() pr.PullRequest {
	return pr.PullRequest{
		Number: 72789,
		Branch: "promote-2-releases-from-qa-to-production-1401301d-1454-41ac-b052-254b21054a8e",
		Labels: []string{"environment:production", "release:release-a", "release:release-b", RenderingFailedLabel},
		Body:   "\nTriggered by: john.doe -> b724eda4e96a93417e54dc8610e0f413bc395c44\n\n# Promotions\n\n# Rendering Errors: boom",
	}
}

func TestParsePullRequest(t *testing.T) {
	promotion, err := ParsePullRequest(newPromotionPullRequest())
	require.NoError(t, err)
	require.Equal(t, &PullRequestPromotion{
		Source:            "qa",
		Target:            "production",
		Releases:          []string{"release-a", "release-b"},
		TemplateVariables: map[string]string{"triggeredBy": "john.doe -> b724eda4e96a93417e54dc8610e0f413bc395c44"},
		Reviewers:         []string{"john.doe"},
	}, promotion)
}

func TestParsePullRequestWithoutTriggeredBy(t *testing.T) {
	pullRequest := newPromotionPullRequest()
	pullRequest.Body = "# Promotions"

	promotion, err := ParsePullRequest(pullRequest)
	require.NoError(t, err)
	require.Nil(t, promotion.TemplateVariables)
	require.Nil(t, promotion.Reviewers)
}

func TestParsePullRequestWithoutLabels(t *testing.T) {
	pullRequest := newPromotionPullRequest()
	pullRequest.Labels = []string{"environment:production"}

	_, err := ParsePullRequest(pullRequest)
	require.EqualError(t, err, "no environment and release labels found")
}

func TestParsePullRequestWithUnexpectedBranch(t *testing.T) {
	pullRequest := newPromotionPullRequest()
	pullRequest.Branch = "feature-branch"

	_, err := ParsePullRequest(pullRequest)
	require.EqualError(t, err, `cannot infer source environment from branch "feature-branch"`)
}
