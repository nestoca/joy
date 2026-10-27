package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/github"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/pkg/catalog"
)

const failedPullRequestsLimit = 25

// updateFailedPullRequests promotes again the releases of the open and unapproved promotion pull requests matching
// given search, updating each pull request in place. Each promotion is performed against the catalog checked out at
// the pull request base commit, so that it promotes the same versions as originally, and the catalog ref is checked
// out again once done.
func updateFailedPullRequests(cmd *cobra.Command, params PromoteParams, search string, dryRun bool) error {
	cfg := config.FromContext(cmd.Context())
	provider := cmp.Or[pr.PullRequestProvider](params.PullRequest, github.NewPullRequestProvider(cfg.CatalogDir))
	loadCatalog := params.LoadCatalogAt
	if loadCatalog == nil {
		loadCatalog = loadCatalogAt
	}

	pullRequests, err := selectFailedPullRequests(provider, search)
	if err != nil {
		return err
	}
	if len(pullRequests) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "🤷 No open and unapproved promotion pull requests matching search")
		return nil
	}

	var errs error
	for _, pullRequest := range pullRequests {
		fmt.Fprintf(cmd.OutOrStdout(), "🔁 Updating pull request %s (%s)\n", pullRequest.URL, pullRequest.Branch)
		if err := updateFailedPullRequest(cmd, params, cfg, loadCatalog, pullRequest, dryRun); err != nil {
			errs = errors.Join(errs, fmt.Errorf("updating pull request %s: %w", pullRequest.URL, err))
		}
	}

	catalogRef := config.CatalogRefFromContext(cmd.Context())
	if _, _, err := loadCatalog(cmd.Context(), cfg, catalogRef, cmd.ErrOrStderr()); err != nil {
		errs = errors.Join(errs, fmt.Errorf("checking out catalog ref %s: %w", catalogRef, err))
	}
	return errs
}

// selectFailedPullRequests returns the open pull requests matching given search, skipping approved and
// non-promotion ones.
func selectFailedPullRequests(provider pr.PullRequestProvider, search string) ([]pr.PullRequest, error) {
	pullRequests, err := provider.List(search, failedPullRequestsLimit)
	if err != nil {
		return nil, err
	}

	var result []pr.PullRequest
	for _, pullRequest := range pullRequests {
		if !pullRequest.Approved && strings.HasPrefix(pullRequest.Branch, promote.BranchPrefix) {
			result = append(result, pullRequest)
		}
	}
	return result, nil
}

func updateFailedPullRequest(cmd *cobra.Command, params PromoteParams, cfg *config.Config, loadCatalog catalogLoaderFunc, pullRequest pr.PullRequest, dryRun bool) error {
	promotion, err := promote.ParsePullRequest(pullRequest)
	if err != nil {
		return err
	}

	baseRef, err := pullRequestBaseRef(pullRequest)
	if err != nil {
		return err
	}

	cfg, cat, err := loadCatalog(cmd.Context(), cfg, baseRef, cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, promotion.Source)
	if err != nil {
		return err
	}

	targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, promotion.Target)
	if err != nil {
		return err
	}

	_, err = newPromotion(cmd, params, cfg, promotion.TemplateVariables).Promote(promote.Opts{
		Catalog:           cat,
		SourceEnv:         sourceEnv,
		TargetEnv:         targetEnv,
		Releases:          promotion.Releases,
		ReleasesFiltered:  true,
		NoPrompt:          true,
		DryRun:            dryRun,
		Reviewers:         promotion.Reviewers,
		UpdatePullRequest: strconv.Itoa(pullRequest.Number),
	})
	return err
}

// pullRequestBaseRef returns the git ref of the catalog commit given pull request was originally created from, which
// is the parent of its first commit, as updates reset its branch on top of that same commit.
func pullRequestBaseRef(pullRequest pr.PullRequest) (string, error) {
	if len(pullRequest.Commits) == 0 {
		return "", fmt.Errorf("pull request has no commits")
	}
	return pullRequest.Commits[0] + "^", nil
}

type catalogLoaderFunc = func(ctx context.Context, cfg *config.Config, ref string, out io.Writer) (*config.Config, *catalog.Catalog, error)

// loadCatalogAt checks out the catalog at given git ref, ensuring it has no uncommitted changes, and loads its
// config and catalog.
func loadCatalogAt(ctx context.Context, cfg *config.Config, ref string, out io.Writer) (*config.Config, *catalog.Catalog, error) {
	if err := git.EnsureCleanAndUpToDateWorkingCopy(cfg.CatalogDir, ref, out); err != nil {
		return nil, nil, fmt.Errorf("checking out catalog at %s: %w", ref, err)
	}

	cfg, err := config.Load(ctx, filepath.Dir(cfg.FilePath), cfg.CatalogDir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading config at %s: %w", ref, err)
	}

	cat, err := catalog.Load(ctx, cfg.CatalogDir, cfg.KnownChartRefs())
	if err != nil {
		return nil, nil, fmt.Errorf("loading catalog at %s: %w", ref, err)
	}

	return cfg, cat, nil
}
