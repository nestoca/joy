//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package promote

type GitProvider interface {
	CreateAndPushBranchWithFiles(branchName string, files []string, message string) error
	CheckoutMasterBranch() error
}
