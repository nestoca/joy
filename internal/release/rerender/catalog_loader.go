package rerender

import (
	"context"

	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/pkg/catalog"
)

//go:generate moq -stub -out ./catalog_loader_mock.go . CatalogLoader
type CatalogLoader interface {
	// LoadAtPullRequestBase returns the catalog as it was when given pull request was created, and a cleanup function.
	LoadAtPullRequestBase(ctx context.Context, pullRequest *pr.PullRequest) (*catalog.Catalog, func(), error)
}
