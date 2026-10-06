package rerender

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/promote"
)

type ReRenderer struct {
	PullRequestProvider pr.PullRequestProvider
	CatalogLoader       CatalogLoader

	// NewPromotion returns the promotion used for re-rendering, configured with given template variables.
	NewPromotion func(variables map[string]string) *promote.Promotion

	// Label restricts the pull requests to re-render when scanning to those with this label. Empty disables it.
	Label string

	// BodyContains restricts the pull requests to re-render when scanning to those whose body contains this text.
	// Empty disables it.
	BodyContains string

	// DryRun prints the changes that would be applied to pull requests instead of applying them.
	DryRun bool

	Out io.Writer
}

// ReRender re-renders the title, body, reviewers and labels of the promotion pull request referenced by given
// number or URL, or of all open and unapproved promotion pull requests with the configured label when ref is empty.
func (r *ReRenderer) ReRender(ctx context.Context, ref string) error {
	pullRequests, err := r.selectPullRequests(ref)
	if err != nil {
		return err
	}

	if len(pullRequests) == 0 {
		_, _ = fmt.Fprintln(r.Out, "🤷 No open and unapproved promotion pull requests matching filters")
		return nil
	}

	var errs error
	for i := range pullRequests {
		errs = errors.Join(errs, r.reRender(ctx, &pullRequests[i]))
	}
	return errs
}

func (r *ReRenderer) selectPullRequests(ref string) ([]pr.PullRequest, error) {
	if ref != "" {
		number, repo, err := ParsePullRequestRef(ref)
		if err != nil {
			return nil, err
		}
		pullRequest, err := r.PullRequestProvider.Get(number)
		if err != nil {
			return nil, err
		}
		if repo != "" {
			_, catalogRepo, err := ParsePullRequestRef(pullRequest.URL)
			if err != nil {
				return nil, fmt.Errorf("parsing catalog pull request url: %w", err)
			}
			if !strings.EqualFold(repo, catalogRepo) {
				return nil, fmt.Errorf("pull request %s is not in catalog repository %s", ref, catalogRepo)
			}
		}
		return []pr.PullRequest{*pullRequest}, nil
	}

	summaries, err := r.PullRequestProvider.ListOpen(r.Label, r.BodyContains)
	if err != nil {
		return nil, err
	}

	var pullRequests []pr.PullRequest
	for _, summary := range summaries {
		if summary.Approved || !strings.HasPrefix(summary.Branch, "promote-") {
			continue
		}
		pullRequest, err := r.PullRequestProvider.Get(summary.Number)
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, *pullRequest)
	}
	return pullRequests, nil
}

func (r *ReRenderer) reRender(ctx context.Context, pullRequest *pr.PullRequest) error {
	_, _ = fmt.Fprintf(r.Out, "🔁 Re-rendering pull request #%d (%s)\n", pullRequest.Number, pullRequest.Branch)

	metadata, err := ResolveMetadata(pullRequest)
	if err != nil {
		return fmt.Errorf("resolving metadata of pull request #%d: %w", pullRequest.Number, err)
	}

	// This is what does the heavy lifting since the versions of the source and target are in the
	// literal state of the repository at this point in time.
	cat, cleanup, err := r.CatalogLoader.LoadAtPullRequestBase(ctx, pullRequest)
	if err != nil {
		return fmt.Errorf("loading catalog for pull request #%d: %w", pullRequest.Number, err)
	}
	defer cleanup()

	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, metadata.Source)
	if err != nil {
		return fmt.Errorf("pull request #%d: %w", pullRequest.Number, err)
	}

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, metadata.Target)
	if err != nil {
		return fmt.Errorf("pull request #%d: %w", pullRequest.Number, err)
	}

	_, err = r.NewPromotion(metadata.Variables).Promote(promote.Opts{
		Catalog:           cat,
		SourceEnv:         sourceEnv,
		TargetEnv:         targetEnv,
		Releases:          metadata.Releases,
		ReleasesFiltered:  true,
		NoPrompt:          true,
		DryRun:            r.DryRun,
		Reviewers:         metadata.Reviewers,
		UpdatePullRequest: pullRequest,
	})
	return err
}
