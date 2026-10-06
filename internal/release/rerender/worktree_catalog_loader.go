package rerender

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/pkg/catalog"
)

type WorktreeCatalogLoader struct {
	dir            string
	knownChartRefs []string
	errOut         io.Writer
}

// NewWorktreeCatalogLoader returns a catalog loader checking out the catalog at given dir in temporary worktrees,
// reporting cleanup failures to errOut.
func NewWorktreeCatalogLoader(dir string, knownChartRefs []string, errOut io.Writer) *WorktreeCatalogLoader {
	return &WorktreeCatalogLoader{dir: dir, knownChartRefs: knownChartRefs, errOut: errOut}
}

// LoadAtPullRequestBase loads the catalog in a temporary worktree checked out at the parent of the pull request's
// first commit, which is the catalog state the promotion was originally rendered from.
func (l *WorktreeCatalogLoader) LoadAtPullRequestBase(ctx context.Context, pullRequest *pr.PullRequest) (*catalog.Catalog, func(), error) {
	if len(pullRequest.Commits) == 0 {
		return nil, nil, fmt.Errorf("pull request #%d has no commits", pullRequest.Number)
	}

	if err := git.FetchPullRequestHead(l.dir, pullRequest.Number); err != nil {
		return nil, nil, err
	}

	base := pullRequest.Commits[0] + "^"

	// The worktree keeps the catalog directory name, as release link templates may rely on it (eg: "^.*/catalog/").
	name := filepath.Join(fmt.Sprintf("re-render-pr-%d", pullRequest.Number), filepath.Base(l.dir))
	dir, remove, err := git.CheckoutWorktree(l.dir, name, base)
	if err != nil {
		return nil, nil, fmt.Errorf("checking out catalog at base of pull request #%d: %w", pullRequest.Number, err)
	}
	cleanup := func() {
		if err := remove(); err != nil {
			_, _ = fmt.Fprintf(l.errOut, "⚠️ Failed to remove catalog worktree %s: %v\n", dir, err)
		}
	}

	cat, err := catalog.Load(ctx, dir, l.knownChartRefs)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("loading catalog at %s: %w", base, err)
	}

	return cat, cleanup, nil
}
