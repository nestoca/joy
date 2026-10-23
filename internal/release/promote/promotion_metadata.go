package promote

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// PromotionMetadata captures what is needed to re-render a promotion pull request after its creation.
// It is embedded in the pull request body as an invisible HTML comment.
// This is effectively a DTO/stripped down version for a promotion info.
type PromotionMetadata struct {
	Source    string            `json:"source"`
	Target    string            `json:"target"`
	Releases  []string          `json:"releases"`
	Variables map[string]string `json:"variables,omitempty"`
	Reviewers []string          `json:"reviewers,omitempty"`
}

// json.Marshal escapes '>' as \u003e, so the encoded metadata can never terminate the comment early.
var metadataRegex = regexp.MustCompile(`<!-- joy-promotion: (.*?) -->`)

// AppendPromotionMetadata returns the given pull request body with the metadata appended as an HTML comment.
func AppendPromotionMetadata(body string, metadata PromotionMetadata) string {
	data, _ := json.Marshal(metadata)
	return fmt.Sprintf("%s\n\n<!-- joy-promotion: %s -->", body, data)
}

// ParsePromotionMetadata extracts the metadata from a pull request body, returning nil if there is none.
func ParsePromotionMetadata(body string) (*PromotionMetadata, error) {
	match := metadataRegex.FindStringSubmatch(body)
	if match == nil {
		return nil, nil
	}

	var metadata PromotionMetadata
	if err := json.Unmarshal([]byte(match[1]), &metadata); err != nil {
		return nil, fmt.Errorf("parsing promotion metadata: %w", err)
	}
	return &metadata, nil
}
