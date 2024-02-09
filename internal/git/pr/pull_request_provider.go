//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package pr

type CreateParams struct {
	Branch string
	Title  string
	Body   string
	Labels []string
	Draft  bool
}

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

	// GetPromotionEnvironment returns the environment to promote builds of given branch's pull request to.
	// If empty string is returned, promotion is disabled.
	GetPromotionEnvironment(branch string) (string, error)

	// SetPromotionEnvironment sets the environment to promote builds of given branch's pull request to.
	// Pass empty string to disable promotion.
	SetPromotionEnvironment(branch, env string) error
}
