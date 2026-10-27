package promote

//go:generate moq -stub -out ./git_provider_mock.go . GitProvider
type GitProvider interface {
	CreateAndPushBranchWithFiles(branchName string, files []string, message string) error
	// ResetAndForcePushBranchWithFiles replaces the content of an existing remote branch with a single commit of
	// given files on top of the current commit.
	ResetAndForcePushBranchWithFiles(branchName string, files []string, message string) error
	CheckoutCatalogRef() error
}
