package promote

//go:generate moq -stub -out ./git_provider_mock.go . GitProvider
type GitProvider interface {
	CreateAndPushBranchWithFiles(branchName string, files []string, message string) error
	// ResetAndForcePushBranchWithFiles replaces the content of an existing remote branch with a single commit of
	// given files on top of the current commit, failing if the remote branch is no longer at given expected head.
	ResetAndForcePushBranchWithFiles(branchName string, expectedHead string, files []string, message string) error
	// DiscardChanges discards uncommitted changes and untracked files left in the catalog.
	DiscardChanges() error
	CheckoutCatalogRef() error
}
