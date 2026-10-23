package promote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataRoundTrip(t *testing.T) {
	metadata := PromotionMetadata{
		Source:    "qa",
		Target:    "production",
		Releases:  []string{"release-a", "release-b"},
		Variables: map[string]string{"triggeredBy": "john.doe -> abc123 -->"},
		Reviewers: []string{"jane.doe"},
	}

	body := AppendPromotionMetadata("# Promotions\nsome body", metadata)

	parsed, err := ParsePromotionMetadata(body)
	require.NoError(t, err)
	require.Equal(t, &metadata, parsed)
}

func TestAppendMetadataIsInvisibleComment(t *testing.T) {
	body := AppendPromotionMetadata("body", PromotionMetadata{Source: "qa", Target: "production"})

	require.Regexp(t, `^body\n\n<!-- joy-promotion: \{.*\} -->$`, body)
}

func TestParseMetadataMissing(t *testing.T) {
	parsed, err := ParsePromotionMetadata("# Promotions\nno metadata here")
	require.NoError(t, err)
	require.Nil(t, parsed)
}

func TestParseMetadataMalformed(t *testing.T) {
	_, err := ParsePromotionMetadata("body\n\n<!-- joy-promotion: {not json} -->")
	require.ErrorContains(t, err, "parsing promotion metadata")
}
