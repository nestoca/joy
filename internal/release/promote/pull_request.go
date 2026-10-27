package promote

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nestoca/joy/internal/git/pr"
)

var (
	triggeredByRegex = regexp.MustCompile(`(?m)^Triggered by: (.+)$`)
	uuidSuffix       = `-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
)

// PullRequestPromotion is the promotion an existing promotion pull request was created from.
type PullRequestPromotion struct {
	Source            string
	Target            string
	Releases          []string
	TemplateVariables map[string]string
	Reviewers         []string
}

// ParsePullRequest infers the promotion given pull request was created from, using its environment and release
// labels, its branch name for the source environment and the "Triggered by" line of its body, if any.
func ParsePullRequest(pullRequest pr.PullRequest) (*PullRequestPromotion, error) {
	var promotion PullRequestPromotion
	for _, label := range pullRequest.Labels {
		if env, ok := strings.CutPrefix(label, "environment:"); ok {
			promotion.Target = env
		}
		if release, ok := strings.CutPrefix(label, "release:"); ok {
			promotion.Releases = append(promotion.Releases, release)
		}
	}
	if promotion.Target == "" || len(promotion.Releases) == 0 {
		return nil, fmt.Errorf("no environment and release labels found")
	}

	branchRegex := regexp.MustCompile(`^promote-.+-from-(.+)-to-` + regexp.QuoteMeta(promotion.Target) + uuidSuffix)
	match := branchRegex.FindStringSubmatch(pullRequest.Branch)
	if match == nil {
		return nil, fmt.Errorf("cannot infer source environment from branch %q", pullRequest.Branch)
	}
	promotion.Source = match[1]

	if match := triggeredByRegex.FindStringSubmatch(pullRequest.Body); match != nil {
		triggeredBy := strings.TrimSpace(match[1])
		promotion.TemplateVariables = map[string]string{"triggeredBy": triggeredBy}

		// The promote workflow passes the commit author both in triggeredBy and as an extra reviewer.
		if author, _, ok := strings.Cut(triggeredBy, " -> "); ok {
			promotion.Reviewers = []string{author}
		}
	}

	return &promotion, nil
}
