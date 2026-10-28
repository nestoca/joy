package pr

type CreateParams struct {
	Branch    string
	Title     string
	Body      string
	Labels    []string
	Reviewers []string
	Draft     bool
}

type UpdateParams struct {
	// PullRequest is the number, URL or branch of the pull request to update.
	PullRequest  string
	Title        string
	Body         string
	Labels       []string
	RemoveLabels []string
	Reviewers    []string
}

type PullRequest struct {
	Number   int
	URL      string
	Branch   string
	Body     string
	Labels   []string
	Open     bool
	Approved bool
	// Commits are the SHAs of the pull request commits, oldest first.
	Commits []string
	// HeadSHA is the SHA of the commit at the head of the pull request branch.
	HeadSHA string
	// CrossRepository is whether the pull request branch is in a fork rather than in the repository itself.
	CrossRepository bool
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

	// Get returns the pull request referenced by given number, URL or branch, failing if it does not belong to the
	// repository.
	Get(pullRequest string) (*PullRequest, error)

	// List returns up to limit pull requests in any state matching given search query, most recent first.
	List(search string, limit int) ([]PullRequest, error)

	// FetchHead fetches the commits of the pull request with given number into the local repository.
	FetchHead(number int) error

	// AddLabels adds given labels to the pull request referenced by given number, URL or branch.
	AddLabels(pullRequest string, labels []string) error

	// Update updates the title and body of an existing pull request, adding the given labels and reviewers, then
	// removes the given labels only once that succeeded, and returns its URL.
	Update(UpdateParams) (string, error)

	// GetPromotionEnvironment returns the environment to promote builds of given branch's pull request to.
	// If empty string is returned, promotion is disabled.
	GetPromotionEnvironment(branch string) (string, error)

	// SetPromotionEnvironment sets the environment to promote builds of given branch's pull request to.
	// Pass empty string to disable promotion.
	SetPromotionEnvironment(branch, env string) error
}
