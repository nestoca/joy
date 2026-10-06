package rerender

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/promote"
)

var (
	triggeredByRegex = regexp.MustCompile(`(?m)^Triggered by: (.+)$`)
	uuidSuffix       = `-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
)

// ResolveMetadata returns the promotion metadata embedded in the pull request body or, for pull requests
// created before metadata was embedded, infers it from the labels, branch name and body.
func ResolveMetadata(pullRequest *pr.PullRequest) (*promote.PromotionMetadata, error) {
	metadata, err := promote.ParsePromotionMetadata(pullRequest.Body)
	if err != nil || metadata != nil {
		return metadata, err
	}

	// Failover code path if there was no PrMetadata embedded in the pull request body.
	// This is for backwards compatibility with promotion pull requests
	metadata = &promote.PromotionMetadata{}
	for _, label := range pullRequest.Labels {
		if env, ok := strings.CutPrefix(label, "environment:"); ok {
			metadata.Target = env
		}
		if release, ok := strings.CutPrefix(label, "release:"); ok {
			metadata.Releases = append(metadata.Releases, release)
		}
	}
	if metadata.Target == "" || len(metadata.Releases) == 0 {
		return nil, fmt.Errorf("no promotion metadata nor environment and release labels found")
	}

	branchRegex := regexp.MustCompile(`^promote-.+-from-(.+)-to-` + regexp.QuoteMeta(metadata.Target) + uuidSuffix)
	match := branchRegex.FindStringSubmatch(pullRequest.Branch)
	if match == nil {
		return nil, fmt.Errorf("cannot infer source environment from branch %q", pullRequest.Branch)
	}
	metadata.Source = match[1]

	if match := triggeredByRegex.FindStringSubmatch(pullRequest.Body); match != nil {
		triggeredBy := strings.TrimSpace(match[1])
		metadata.Variables = map[string]string{"triggeredBy": triggeredBy}

		// The promote workflow passes the commit author both in triggeredBy and as an extra reviewer.
		if author, _, ok := strings.Cut(triggeredBy, " -> "); ok {
			metadata.Reviewers = []string{author}
		}
	}

	return metadata, nil
}
