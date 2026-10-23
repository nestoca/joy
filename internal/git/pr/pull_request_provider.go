package pr

import (
	"fmt"
	"slices"
	"strings"
)

var nonRequestableReviewers = []string{
	"",
	"*",
	"invalid-email-address",
	"nestobot",
	"copilot",
	"claude",
	"cursor",
}

// IsRequestableReviewer returns whether a review can be requested from given user.
// Bots cannot be requested as reviewers (GitHub returns "not found"), nor can placeholder or AI agent authors.
func IsRequestableReviewer(reviewer string) bool {
	lower := strings.ToLower(reviewer)
	return !strings.HasSuffix(lower, "[bot]") && !slices.Contains(nonRequestableReviewers, lower)
}

type CreateParams struct {
	Branch    string
	Title     string
	Body      string
	Labels    []string
	Reviewers []string
	Draft     bool
}

type PullRequest struct {
	Number int
	URL    string
	Branch string
	Title  string
	Body   string
	Labels []string
	// Approved indicates the pull request review decision is approved.
	Approved bool
	// Reviewers are the users whose review was requested or who already reviewed.
	Reviewers []string
	// Commits are the SHAs of the pull request commits, oldest first.
	Commits []string
}

// Link returns the URL of the pull request, or its number when its URL is unknown.
func (p PullRequest) Link() string {
	if p.URL == "" {
		return fmt.Sprintf("#%d", p.Number)
	}
	return p.URL
}

// UpdateParams describes changes to an existing pull request. Empty fields are left unchanged.
type UpdateParams struct {
	Number       int
	Title        string
	Body         string
	AddReviewers []string
	AddLabels    []string
	RemoveLabels []string
}

// IsEmpty returns whether the update would leave the pull request unchanged.
func (p UpdateParams) IsEmpty() bool {
	return p.Title == "" && p.Body == "" && len(p.AddReviewers) == 0 && len(p.AddLabels) == 0 && len(p.RemoveLabels) == 0
}

//go:generate moq -stub -out ./pull_request_provider_mock.go . PullRequestProvider
type PullRequestProvider interface {
	// EnsureInstalledAndAuthenticated ensures the service provider is installed and authorized.
	EnsureInstalledAndAuthenticated() error

	// Exists returns whether a pull request exists for given branch.
	Exists(branch string) (bool, error)

	// GetBranchesPromotingToEnvironment returns the branches with pull requests configured for auto-promotion
	// to given environment.
	GetBranchesPromotingToEnvironment(env string) ([]string, error)

	// CreateInteractively prompts user to create a pull request for given branch.
	CreateInteractively(branch string) error

	// Create creates a pull request for given branch
	Create(CreateParams) (string, error)

	// Get returns the pull request with given number.
	Get(number int) (*PullRequest, error)

	// List returns up to limit pull requests in any state, most recent first, with given label and whose body
	// contains given text. Empty label or text means no filtering on it.
	// Only the number, URL, branch, labels and approval are populated, use Get for full details.
	List(label string, bodyContains string, limit int) ([]PullRequest, error)

	// FetchHead fetches the commits of the pull request with given number into the local repository.
	FetchHead(number int) error

	// Update applies given changes to an existing pull request.
	Update(UpdateParams) error

	// GetPromotionEnvironment returns the environment to promote builds of given branch's pull request to.
	// If empty string is returned, promotion is disabled.
	GetPromotionEnvironment(branch string) (string, error)

	// SetPromotionEnvironment sets the environment to promote builds of given branch's pull request to.
	// Pass empty string to disable promotion.
	SetPromotionEnvironment(branch, env string) error
}
