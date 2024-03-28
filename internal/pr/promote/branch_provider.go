package promote

//go:generate moq -stub -out ./branch_provider_mock.go . BranchProvider
type BranchProvider interface {
	// GetCurrentBranch returns name of branch currently checked out in current working directory.
	GetCurrentBranch() (string, error)
}
