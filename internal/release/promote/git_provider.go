package promote

//go:generate moq -stub -out ./git_provider_mock.go . GitProvider
type GitProvider interface {
	CreateAndPushBranchWithFiles(branchName string, files []string, message string) error
	CheckoutMasterBranch() error
}
