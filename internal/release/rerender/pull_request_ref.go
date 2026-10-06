package rerender

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var pullRequestPathRegex = regexp.MustCompile(`^/([^/]+/[^/]+)/pull/(\d+)(/.*)?$`)

// ParsePullRequestRef parses a pull request number (eg: "123" or "#123") or URL
// (eg: "https://github.com/owner/repo/pull/123"), returning its number and, for URLs, its "owner/repo".
func ParsePullRequestRef(ref string) (number int, repo string, err error) {
	if number, err := strconv.Atoi(strings.TrimPrefix(ref, "#")); err == nil && number > 0 {
		return number, "", nil
	}

	parsed, err := url.Parse(ref)
	if err == nil && parsed.Host != "" {
		if match := pullRequestPathRegex.FindStringSubmatch(parsed.Path); match != nil {
			number, _ := strconv.Atoi(match[2])
			return number, match[1], nil
		}
	}

	return 0, "", fmt.Errorf("invalid pull request %q: expecting a number or a pull request url", ref)
}
