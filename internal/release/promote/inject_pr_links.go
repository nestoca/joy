package promote

import (
	"fmt"
	"regexp"
)

var pullRequestReferenceRegex = regexp.MustCompile(`(?m)(^|\s)#(\d+)\b`)

func injectPullRequestLinks(repo string, text string) (string, error) {
	// Iterate over the matches in reverse order, to prevent replacement from offsetting indexes
	matches := pullRequestReferenceRegex.FindAllStringSubmatchIndex(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		prefix := text[match[2]:match[3]]
		prNumber := text[match[4]:match[5]]
		replacement := fmt.Sprintf("[#%s](https://github.com/%s/pull/%s)", prNumber, repo, prNumber)
		text = text[:match[0]] + prefix + replacement + text[match[1]:]
	}

	return text, nil
}
