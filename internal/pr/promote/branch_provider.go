//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package promote

type BranchProvider interface {
	// GetCurrentBranch returns name of branch currently checked out in current working directory.
	GetCurrentBranch() (string, error)
}
