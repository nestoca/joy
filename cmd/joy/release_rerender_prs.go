package main

import (
	"cmp"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/catalog/loader"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/github"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/release/rerender"
	"github.com/nestoca/joy/internal/yml"
)

// NewReleaseReRenderPRsCmd returns the command re-rendering promotion pull requests that failed to render.
func NewReleaseReRenderPRsCmd(params PromoteParams) *cobra.Command {
	var ref string
	var dryRun bool
	var label string
	var bodyContains string
	var limit int

	cmd := &cobra.Command{
		Use:   "re-render-prs",
		Short: "Re-render promotion pull requests that failed to render",
		Long: fmt.Sprintf(`Re-render the title, body and reviewers of unapproved promotion pull requests labeled %s
(see --label and --body-contains), in any state, without recreating them. Pull requests are only modified when the
re-rendered content differs.

The catalog is rendered as it was when each pull request was created, using a temporary worktree,
so the local catalog working copy is left untouched.`, promote.RenderingFailedLabel),
		Example: `  # Re-render all promotion pull requests that failed to render
  joy release re-render-prs

  # Re-render a specific promotion pull request
  joy release re-render-prs --pr 72789
  joy release re-render-prs --pr https://github.com/nestoca/catalog/pull/72789

  # Re-render promotion pull requests whose body contains a given error, regardless of labels
  joy release re-render-prs --label "" --body-contains "No such device or address"

  # Re-render all unapproved promotion pull requests, regardless of labels
  joy release re-render-prs --label ""

  # Preview changes without modifying pull requests
  joy release re-render-prs --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			pullRequestProvider := cmp.Or[pr.PullRequestProvider](params.PullRequest, github.NewPullRequestProvider(cfg.CatalogDir))

			reRenderer := rerender.ReRenderer{
				PullRequestProvider: pullRequestProvider,
				CatalogLoader:       loader.NewWorktreeLoader(cfg.CatalogDir),
				NewPromotion: func(variables map[string]string) *promote.Promotion {
					return &promote.Promotion{
						CommitTemplate:      cfg.Templates.Release.Promote.Commit,
						PullRequestTemplate: cfg.Templates.Release.Promote.PullRequest,
						TemplateVariables:   variables,
						PromptProvider:      cmp.Or[promote.PromptProvider](params.Prompt, promote.NewInteractivePromptProvider(cmd.OutOrStdout())),
						GitProvider:         cmp.Or[promote.GitProvider](params.Git, promote.NewShellGitProvider(cfg.CatalogDir)),
						PullRequestProvider: pullRequestProvider,
						YamlWriter:          cmp.Or[yml.Writer](params.Writer, yml.DiskWriter),
						InfoProvider:        cmp.Or(params.Info, infoProvider),
						LinksProvider:       cmp.Or(params.Links, links.NewProvider(infoProvider, cfg.Templates)),
						Out:                 cmd.OutOrStdout(),
					}
				},
				Label:        label,
				BodyContains: bodyContains,
				Limit:        limit,
				DryRun:       dryRun,
				Out:          cmd.OutOrStdout(),
			}

			return reRenderer.ReRender(cmd.Context(), ref)
		},
	}

	cmd.Flags().StringVar(&ref, "pr", "", "Number or URL of a specific promotion pull request to re-render, regardless of its labels, approval and state")
	cmd.Flags().StringVar(&label, "label", promote.RenderingFailedLabel, "Label the pull requests to re-render must have when scanning (empty to disable)")
	cmd.Flags().StringVar(&bodyContains, "body-contains", "", "Text the body of the pull requests to re-render must contain when scanning (empty to disable).")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of most recent pull requests matching the filters to consider when scanning")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the changes that would be applied to pull requests without applying them")

	return cmd
}
