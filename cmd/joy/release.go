package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"gopkg.in/yaml.v3"

	"github.com/TwiN/go-color"
	"github.com/davidmdm/x/xerr"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/github"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/release/list"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/release/validate"
	"github.com/nestoca/joy/internal/text"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
	"github.com/nestoca/joy/pkg/helm"
)

func NewReleaseCmd(preRunConfigs PreRunConfigs) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel", "r"},
		Short:   "Manage releases",
		Long:    `Manage releases, such as promoting a release in a given environment.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewReleaseListCmd(preRunConfigs))
	cmd.AddCommand(NewReleasePromoteCmd(PromoteParams{PreRunConfigs: preRunConfigs}))
	cmd.AddCommand(NewReleaseSelectCmd(preRunConfigs))
	cmd.AddCommand(NewReleaseRenderCmd())
	cmd.AddCommand(NewReleaseOpenCmd())
	cmd.AddCommand(NewReleaseLinksCmd())
	cmd.AddCommand(NewReleaseSchemaCmd())
	cmd.AddCommand(NewGitCommands())
	cmd.AddCommand(NewValidateCommand())

	return cmd
}

func NewReleaseListCmd(preRunConfigs PreRunConfigs) *cobra.Command {
	var releases, envs, owners string
	var narrow, wide bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "List releases across environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			if releases != "" {
				return fmt.Errorf("--releases flag no longer supported, please specify comma-delimited list of releases as first positional argument")
			}

			if len(args) > 0 {
				releasePattern := args[0]
				cat.WithReleaseFilter(filtering.NewNamePatternFilter(releasePattern))
			} else if len(cfg.Releases.Selected) > 0 {
				cat.WithReleaseFilter(filtering.NewSpecificReleasesFilter(cfg.Releases.Selected))
			}

			selectedEnvs := func() []string {
				if envs == "" {
					return cfg.Environments.Selected
				}
				return strings.Split(envs, ",")
			}()
			if owners != "" {
				cat.WithReleaseFilter(filtering.NewOwnerFilter(owners))
			}

			releaseList, err := list.GetReleaseList(cat, list.Params{
				SelectedEnvs:         selectedEnvs,
				ReferenceEnvironment: cfg.ReferenceEnvironment,
			})
			if err != nil {
				return fmt.Errorf("getting release list: %w", err)
			}

			if jsonOutput {
				output, err := list.FormatReleaseListAsJson(releaseList)
				if err != nil {
					return fmt.Errorf("formatting release list as JSON: %w", err)
				}
				fmt.Println(output)
				return nil
			}

			output := list.FormatReleaseListAsTable(releaseList, cfg.ReferenceEnvironment, cfg.ColumnWidths.Get(narrow, wide))
			fmt.Println(output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to list (comma-separated with wildcards, defaults to configured selection or all)")
	cmd.Flags().StringVarP(&envs, "env", "e", "", "environments to list (comma-separated, defaults to configured selection or all)")
	cmd.Flags().StringVarP(&owners, "owners", "o", "", "List releases by owners (comma-separated, defaults to all)")
	cmd.Flags().BoolVarP(&narrow, "narrow", "n", false, "Use narrow columns mode")
	cmd.Flags().BoolVarP(&wide, "wide", "w", false, "Use wide columns mode")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.MarkFlagsMutuallyExclusive("narrow", "wide")

	preRunConfigs.PullCatalog(cmd)

	return cmd
}

type PromoteParams struct {
	Links         links.Provider
	Info          info.Provider
	Git           promote.GitProvider
	PullRequest   pr.PullRequestProvider
	Prompt        promote.PromptProvider
	Writer        yml.Writer
	PreRunConfigs PreRunConfigs
}

func NewReleasePromoteCmd(params PromoteParams) *cobra.Command {
	var sourceEnv, targetEnv string
	var autoMerge, draft, dryRun, localOnly, noPrompt, narrow, wide bool
	var all, keepPrerelease bool
	var omit []string
	var templateVars []string
	var reviewers []string

	cmd := &cobra.Command{
		Use:     "promote [flags] [releases]",
		Aliases: []string{"prom", "p"},
		Short:   "Promote releases across environments",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if autoMerge && draft {
				return fmt.Errorf("flags --auto-merge and --draft cannot be used together")
			}
			if noPrompt {
				if len(args) == 0 && !all {
					return fmt.Errorf("one of releases or --all are required when no-prompt is set")
				}
				if sourceEnv == "" {
					return fmt.Errorf("source environment is required when no-prompt is set")
				}
				if targetEnv == "" {
					return fmt.Errorf("target environment is required when no-prompt is set")
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, releases []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			var filter filtering.Filter
			if len(releases) == 0 && !all && len(cfg.Releases.Selected) > 0 {
				// if there is no pre-selection, ie: user did not explicity pass releases nor use the --all flag
				// we want to limit the catalog releases to the user config defined release selection.
				cat.WithReleaseFilter(filtering.NewSpecificReleasesFilter(cfg.Releases.Selected))
			}

			sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, sourceEnv)
			if err != nil {
				return err
			}

			targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, targetEnv)
			if err != nil {
				return err
			}

			templateVariables, err := parseTemplateVars(templateVars)
			if err != nil {
				return err
			}

			selectedEnvironments := v1alpha1.GetEnvironmentsByNames(cat.Environments, cfg.Environments.Selected)

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)

			promoter := promote.Promotion{
				CommitTemplate:      cfg.Templates.Release.Promote.Commit,
				PullRequestTemplate: cfg.Templates.Release.Promote.PullRequest,
				TemplateVariables:   templateVariables,
				PromptProvider:      cmp.Or[promote.PromptProvider](params.Prompt, promote.NewInteractivePromptProvider(cmd.OutOrStdout())),
				GitProvider:         cmp.Or[promote.GitProvider](params.Git, promote.NewShellGitProvider(cfg.CatalogDir)),
				PullRequestProvider: cmp.Or[pr.PullRequestProvider](params.PullRequest, github.NewPullRequestProvider(cfg.CatalogDir)),
				YamlWriter:          cmp.Or[yml.Writer](params.Writer, yml.DiskWriter),
				InfoProvider:        cmp.Or(params.Info, infoProvider),
				LinksProvider:       cmp.Or(params.Links, links.NewProvider(infoProvider, cfg.Templates)),
				Out:                 cmd.OutOrStdout(),
			}

			opts := promote.Opts{
				Catalog:              cat,
				SourceEnv:            sourceEnv,
				TargetEnv:            targetEnv,
				Releases:             releases,
				ReleasesFiltered:     len(releases) > 0 || filter != nil,
				NoPrompt:             noPrompt,
				AutoMerge:            autoMerge,
				All:                  all,
				Omit:                 omit,
				KeepPrerelease:       keepPrerelease,
				Draft:                draft,
				SelectedEnvironments: selectedEnvironments,
				DryRun:               dryRun,
				LocalOnly:            localOnly,
				MaxColumnWidth:       cfg.ColumnWidths.Get(narrow, wide),
				Reviewers:            reviewers,
			}

			_, err = promoter.Promote(opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&sourceEnv, "source", "s", "", "Source environment (interactive if not specified)")
	cmd.Flags().StringVarP(&targetEnv, "target", "t", "", "Target environment (interactive if not specified)")
	cmd.Flags().BoolVar(&autoMerge, "auto-merge", false, "Add auto-merge label to release PR")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create draft release PR")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run (do not create PR)")
	cmd.Flags().BoolVar(&localOnly, "local-only", false, "Similar to dry-run, but updates the release file(s) on the local filesystem only. There is no branch, commits, or PR created.")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Do not prompt user for anything")
	cmd.Flags().StringSliceVar(&templateVars, "template-var", nil, "Variable to pass to PR/commit templates, in the form of key=value (flag can be specified multiple times)")
	cmd.Flags().BoolVarP(&narrow, "narrow", "n", false, "Use narrow columns mode")
	cmd.Flags().BoolVarP(&wide, "wide", "w", false, "Use wide columns mode")
	cmd.Flags().BoolVar(&all, "all", false, "Select all releases")
	cmd.Flags().BoolVar(&keepPrerelease, "keep-prerelease", false, "Do not promote releases that are prereleases in target env")
	cmd.Flags().StringSliceVar(&omit, "omit", nil, "Releases to omit from promotion")
	cmd.Flags().StringSliceVar(&reviewers, "reviewers", nil, "Additional reviewers to add to the PR (can be specified multiple times)")
	cmd.MarkFlagsMutuallyExclusive("narrow", "wide")

	params.PreRunConfigs.PullCatalog(cmd)

	return cmd
}

func parseTemplateVars(templateVars []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, templateVar := range templateVars {
		key, value, ok := strings.Cut(templateVar, "=")
		if !ok {
			return nil, fmt.Errorf("malformed template variable (expecting 'key=value' pair): %q", templateVar)
		}
		vars[key] = value
	}
	return vars, nil
}

func NewReleaseSelectCmd(preRunConfigs PreRunConfigs) *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Select releases to include in listings and promotions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())
			return release.ConfigureSelection(cat, cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all releases (non-interactive)")

	preRunConfigs.PullCatalog(cmd)

	return cmd
}

func NewReleaseRenderCmd() *cobra.Command {
	var (
		all          bool
		allEnvs      bool
		environments []string
		colorEnabled bool
		gitRef       string
		diffRef      string
		diffContext  int
		verbose      bool
		valuesOnly   bool
		normalize    bool
		debug        bool
	)

	cmd := &cobra.Command{
		Use:   "render [release]",
		Short: "render kubernetes manifests from joy release",
		RunE: func(cmd *cobra.Command, releases []string) (err error) {
			cfg := config.FromContext(cmd.Context())

			uniq := func(values []string) []string {
				dedup := map[string]struct{}{}
				for _, value := range values {
					dedup[value] = struct{}{}
				}
				var result []string
				for key := range dedup {
					result = append(result, key)
				}
				slices.Sort(result)
				return result
			}

			useRef := func(ref string) (restore func() error, err error) {
				if ref == "" {
					return func() error { return nil }, nil
				}

				var restoreFuncs []func() error

				// Stash does not stash untracked files so we do not want a dirty state unless it is stashable
				dirty, err := git.IsDirty(cfg.CatalogDir, git.UncommittedChangesOptions{SkipUntrackedFiles: true})
				if err != nil {
					return nil, fmt.Errorf("checking if catalog is in dirty state: %w", err)
				}

				if dirty {
					if err := git.Stash(cfg.CatalogDir); err != nil {
						return nil, fmt.Errorf("stashing catalog: %w", err)
					}
					restoreFuncs = append(restoreFuncs, func() error {
						if err := git.StashApply(cfg.CatalogDir); err != nil {
							return fmt.Errorf("applying stash: %w", err)
						}
						return nil
					})
				}

				if err := git.Checkout(cfg.CatalogDir, ref); err != nil {
					return nil, fmt.Errorf("checking out: %s: %w", ref, err)
				}
				restoreFuncs = append(restoreFuncs, func() error {
					if err := git.SwitchBack(cfg.CatalogDir); err != nil {
						return fmt.Errorf("switching git back to previous branch: %w", err)
					}
					return nil
				})

				return func() error {
					slices.Reverse(restoreFuncs)
					errs := make([]error, len(restoreFuncs))
					for i, fn := range restoreFuncs {
						errs[i] = fn()
					}
					return xerr.MultiErrFrom("restoring refs", errs...)
				}, nil
			}

			knownEnvironments, err := func() (envs []string, err error) {
				for _, ref := range uniq([]string{gitRef, diffRef}) {
					restore, err := useRef(ref)
					if err != nil {
						return nil, fmt.Errorf("using ref: %s: %w", ref, err)
					}
					defer func() { err = xerr.MultiErrFrom("", err, restore()) }()

					// In this case we cannot use the config or catalog loaded from the context
					// Since we need to reload at whatever git reference we are at.
					cfg, err := config.Load(cmd.Context(), "", cfg.CatalogDir)
					if err != nil {
						return nil, fmt.Errorf("loading config: %w", err)
					}

					cat, err := catalog.Load(cmd.Context(), cfg.CatalogDir, cfg.KnownChartRefs())
					if err != nil {
						return nil, fmt.Errorf("loading catalog: %w", err)
					}

					envs = append(envs, cat.GetEnvironmentNames()...)
				}

				return uniq(envs), nil
			}()
			if err != nil {
				return fmt.Errorf("getting known environments: %w", err)
			}

			if !allEnvs && len(environments) == 0 {
				environments, err = internal.MultiSelect("Which environment(s) do you want to render from?", knownEnvironments)
				if err != nil {
					return
				}
			}
			if allEnvs {
				environments = nil
			}

			var errs []error
			for _, env := range environments {
				if !slices.Contains(knownEnvironments, env) {
					errs = append(errs, fmt.Errorf("%s", env))
				}
			}

			if err := xerr.MultiErrOrderedFrom("unknown environment(s)", errs...); err != nil {
				return err
			}

			knownReleases, err := func() (releases []string, err error) {
				for _, ref := range uniq([]string{gitRef, diffRef}) {
					restore, err := useRef(ref)
					if err != nil {
						return nil, fmt.Errorf("using ref: %s: %w", ref, err)
					}
					defer func() { err = xerr.MultiErrFrom("", err, restore()) }()

					// In this case we cannot use the config or catalog loaded from the context
					// Since we need to reload at whatever git reference we are at.
					cfg, err := config.Load(cmd.Context(), "", cfg.CatalogDir)
					if err != nil {
						return nil, fmt.Errorf("loading config: %w", err)
					}

					cat, err := catalog.Load(cmd.Context(), cfg.CatalogDir, cfg.KnownChartRefs())
					if err != nil {
						return nil, fmt.Errorf("loading catalog: %w", err)
					}

					cat.WithEnvironments(environments)

					releases = append(releases, cat.GetReleaseNames()...)
				}

				return uniq(releases), nil
			}()
			if err != nil {
				return fmt.Errorf("getting known releases: %w", err)
			}

			if !all && len(releases) == 0 {
				releases, err = internal.MultiSelect("Which release(s) do you want to render?", knownReleases)
				if err != nil {
					return
				}
			}
			if all {
				releases = nil
			}

			for _, releaseItem := range releases {
				if !slices.Contains(knownReleases, releaseItem) {
					errs = append(errs, fmt.Errorf("%s", releaseItem))
				}
			}

			if err := xerr.MultiErrOrderedFrom("unknown release(s)", errs...); err != nil {
				return err
			}

			renderRef := func(ref string) (result map[string]string, err error) {
				restore, err := useRef(ref)
				if err != nil {
					return nil, fmt.Errorf("using ref: %s: %w", ref, err)
				}
				defer func() { err = xerr.MultiErrFrom("", err, restore()) }()

				// In this case we cannot use the config or catalog loaded from the context
				// Since we need to reload at whatever git reference we are at.
				cfg, err := config.Load(cmd.Context(), "", cfg.CatalogDir)
				if err != nil {
					return nil, fmt.Errorf("loading config: %w", err)
				}

				cat, err := catalog.Load(cmd.Context(), cfg.CatalogDir, cfg.KnownChartRefs())
				if err != nil {
					return nil, fmt.Errorf("loading catalog: %w", err)
				}

				cat.WithEnvironments(environments)
				cat.WithReleases(releases)

				cache := helm.ChartCache{
					Refs:            cfg.Charts,
					DefaultChartRef: cfg.DefaultChartRef,
					Root:            cfg.JoyCache,
					Puller:          helm.CLI{IO: internal.IoFromCommand(cmd)},
				}

				results := map[string]string{}

				for i := range cat.Releases.Environments {
					for _, cross := range cat.Releases.Items {
						releaseItem := cross.Releases[i]
						if releaseItem == nil {
							continue
						}

						releaseIdentifier := releaseItem.Environment.Name + "/" + releaseItem.Name

						chart, err := cache.GetReleaseChartFS(cmd.Context(), releaseItem)
						if err != nil {
							return nil, fmt.Errorf("getting chart for release: %s: %w", releaseIdentifier, err)
						}

						params := render.RenderParams{
							Release:    releaseItem,
							Chart:      chart,
							Helm:       helm.CLI{IO: internal.IoFromCommand(cmd), Debug: debug},
							ValuesOnly: valuesOnly,
						}

						result, err := render.Render(cmd.Context(), params)
						if err != nil {
							return nil, fmt.Errorf("rendering release: %s: %w", releaseIdentifier, err)
						}

						if normalize {
							var (
								builder strings.Builder
								encoder = yaml.NewEncoder(&builder)
								decoder = yaml.NewDecoder(strings.NewReader(result))
							)
							for {
								var elem any
								if err := decoder.Decode(&elem); err != nil {
									if errors.Is(err, io.EOF) {
										break
									}
									return nil, fmt.Errorf("failed to decode values: %w", err)
								}
								if err := encoder.Encode(elem); err != nil {
									return nil, fmt.Errorf("failed to encode values: %w", err)
								}
							}
							result = builder.String()
						}

						results[releaseIdentifier] = result
					}
				}

				return results, nil
			}

			orderedKeys := func(maps ...map[string]string) []string {
				dedup := map[string]struct{}{}
				for _, m := range maps {
					for key := range m {
						dedup[key] = struct{}{}
					}
				}

				var keys []string
				for key := range dedup {
					keys = append(keys, key)
				}

				slices.Sort(keys)
				return keys
			}

			header := func(name string) string {
				value := fmt.Sprintf("===\n%s\n===", name)
				if colorEnabled {
					value = color.InBold(color.InCyan(value))
				}
				return value
			}

			content := func(value string) string {
				if !colorEnabled {
					return value
				}
				lines := strings.Split(value, "\n")
				for i, line := range lines {
					if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "# Source:") {
						lines[i] = color.InYellow(line)
					}
				}
				return strings.Join(lines, "\n")
			}

			gitRefResult, err := renderRef(gitRef)
			if err != nil {
				return err
			}

			if diffRef == "" {
				for _, key := range orderedKeys(gitRefResult) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n\n", header(key), content(gitRefResult[key]))
				}
				return nil
			}

			diffRefResult, err := renderRef(diffRef)
			if err != nil {
				return err
			}

			diffFunc := func() text.DiffFunc {
				if colorEnabled {
					return text.DiffColorized
				}
				return text.Diff
			}()

			for _, key := range orderedKeys(gitRefResult, diffRefResult) {
				diff := diffFunc(
					text.File{Name: cmp.Or(gitRef, "(current)"), Content: content(gitRefResult[key])},
					text.File{Name: diffRef, Content: content(diffRefResult[key])},
					diffContext,
				)
				if !verbose && diff == "" {
					continue
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n\n", header(key), diff)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&gitRef, "git-ref", "", "git ref to checkout before render")
	cmd.Flags().StringVar(&diffRef, "diff-ref", "", "git ref to checkout before render")
	cmd.Flags().IntVarP(&diffContext, "diff-context", "c", 4, "line context when rendering diff")

	cmd.Flags().StringSliceVarP(&environments, "env", "e", nil, "environments to select releases from.")
	cmd.Flags().BoolVar(&colorEnabled, "color", term.IsTerminal(int(os.Stdout.Fd())), "toggle output with color")
	cmd.Flags().BoolVar(&all, "all", false, "select all releases to be rendered")
	cmd.Flags().BoolVar(&allEnvs, "all-envs", false, "select all environments to render from")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "print empty diffs with headers")
	cmd.Flags().BoolVar(&valuesOnly, "values", false, "print rendered chart values only")
	cmd.Flags().BoolVar(&debug, "debug", false, "send the --debug flag to the helm cli")
	cmd.Flags().BoolVar(&normalize, "normalize", false, "decodes and re-encodes the rendered yaml into a normalized format so that templating diffs are ignored")

	return cmd
}

func NewValidateCommand() *cobra.Command {
	var env string
	var noRender bool

	cmd := &cobra.Command{
		Use:   "validate [releases...]",
		Short: "validate releases",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			selectedEnvs := func() []string {
				if env == "" {
					return cfg.Environments.Selected
				}
				return strings.Split(env, ",")
			}()

			releaseFilter := func() filtering.Filter {
				if len(args) == 0 {
					return nil
				}
				return filtering.NewSpecificReleasesFilter(args)
			}()

			cat := catalog.FromContext(cmd.Context())
			cat.WithEnvironments(selectedEnvs)
			cat.WithReleaseFilter(releaseFilter)

			var releases []*v1alpha1.Release
			for _, item := range cat.Releases.Items {
				for _, rel := range item.Releases {
					if rel == nil {
						continue
					}
					releases = append(releases, rel)
				}
			}

			return validate.Validate(cmd.Context(), validate.ValidateParams{
				Releases: releases,
				NoRender: noRender,
				Helm:     helm.CLI{IO: internal.IO{Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr(), In: cmd.InOrStdin()}},
				ChartCache: helm.ChartCache{
					Refs:            cfg.Charts,
					DefaultChartRef: cfg.DefaultChartRef,
					Root:            cfg.JoyCache,
					Puller:          helm.CLI{IO: internal.IO{Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr(), In: cmd.InOrStdin()}},
				},
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "environment to select release from.")
	cmd.Flags().BoolVarP(&noRender, "no-render", "", false, "skips release rendering validation step")
	return cmd
}

func NewGitCommands() *cobra.Command {
	buildCommand := func(command string) *cobra.Command {
		var (
			source string
			target string
		)
		cmd := &cobra.Command{
			Use:     command + " <release>",
			Aliases: []string{command[0:1]},
			Args:    cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg := config.FromContext(cmd.Context())
				cat := catalog.FromContext(cmd.Context())

				if target == "" {
					target = cfg.ReferenceEnvironment
				}

				if target == "" {
					return fmt.Errorf("unable to determine target environment: specify target or set your reference environment in your config")
				}

				releaseName := args[0]

				var sourceRelease, targetRelease *v1alpha1.Release

				for _, cross := range cat.Releases.Items {
					if cross.Name != releaseName {
						continue
					}
					for _, rel := range cross.Releases {
						if rel == nil {
							continue
						}
						switch rel.Environment.Name {
						case source:
							sourceRelease = rel
						case target:
							targetRelease = rel
						}
					}
					break
				}

				if sourceRelease == nil {
					return fmt.Errorf("no source release found")
				}
				if targetRelease == nil {
					return fmt.Errorf("no target release found")
				}

				infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
				repoDir, err := infoProvider.GetProjectSourceDir(sourceRelease.Project)
				if err != nil {
					return fmt.Errorf("cloning repository: %w", err)
				}

				sourceTag, err := infoProvider.GetReleaseGitTag(sourceRelease)
				if err != nil {
					return fmt.Errorf("getting tag for source version %s of release %s: %w", sourceRelease.Spec.Version, sourceRelease.Name, err)
				}

				targetTag, err := infoProvider.GetReleaseGitTag(targetRelease)
				if err != nil {
					return fmt.Errorf("getting tag for target version %s of release %s: %w", targetRelease.Spec.Version, targetRelease.Name, err)
				}

				gitArgs := append([]string{command}, args[1:]...)
				gitArgs = append(gitArgs, sourceTag+".."+targetTag)

				if sourceRelease.Project.Spec.RepositorySubpaths != nil {
					if !slices.Contains(gitArgs, "--") {
						gitArgs = append(gitArgs, "--")
					}
					gitArgs = append(gitArgs, sourceRelease.Project.Spec.RepositorySubpaths...)
				}

				fmt.Println(color.InCyan("running:"), color.InBold("git "+strings.Join(gitArgs, " ")), color.InCyan("in"), color.InBold(repoDir))
				fmt.Println()

				gitCommand := exec.Command("git", gitArgs...)
				gitCommand.Dir = repoDir
				gitCommand.Stdout = os.Stdout
				gitCommand.Stderr = os.Stderr
				gitCommand.Stdin = os.Stdin
				gitCommand.Env = os.Environ()

				return gitCommand.Run()
			},
		}

		cmd.Flags().StringVar(&source, "source", "", "source environment to compare release from")
		cmd.Flags().StringVar(&target, "target", "", "target environment to compare release to")

		if err := cmd.MarkFlagRequired("source"); err != nil {
			panic(err)
		}

		return cmd
	}

	root := &cobra.Command{
		Use:     "git",
		Aliases: []string{"g"},
		Short:   "apply git commands to releases between environments",
	}

	root.AddCommand(buildCommand("diff"))
	root.AddCommand(buildCommand("log"))

	return root
}

func NewReleaseOpenCmd() *cobra.Command {
	var env string

	cmd := &cobra.Command{
		Use:     "open [flags] [release] [link]",
		Aliases: []string{"open", "o"},
		Short:   "Open release link",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			releaseName := ""
			if len(args) >= 1 {
				releaseName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			var filter filtering.Filter
			if releaseName == "" && len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			cat.WithReleaseFilter(filter)

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			releaseLinks, err := links.GetReleaseLinks(linksProvider, cat, env, releaseName)
			if err != nil {
				return fmt.Errorf("getting release links: %w", err)
			}

			url, err := links.GetOrSelectLinkUrl(releaseLinks, linkName)
			if err != nil {
				return err
			}

			return browser.OpenURL(url)
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (interactive if not specified)")

	return cmd
}

func NewReleaseLinksCmd() *cobra.Command {
	var env string

	cmd := &cobra.Command{
		Use:     "links [flags] [release] [link]",
		Aliases: []string{"links", "link", "lnk"},
		Short:   "List release links",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			cat := catalog.FromContext(cmd.Context())

			releaseName := ""
			if len(args) >= 1 {
				releaseName = args[0]
			}

			linkName := ""
			if len(args) >= 2 {
				linkName = args[1]
			}

			var filter filtering.Filter
			if releaseName == "" && len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			cat.WithReleaseFilter(filter)

			infoProvider := info.NewProvider(cfg.GitHubOrganization, cfg.Templates.Project.GitTag, cfg.RepositoriesDir, cfg.JoyCache)
			linksProvider := links.NewProvider(infoProvider, cfg.Templates)

			releaseLinks, err := links.GetReleaseLinks(linksProvider, cat, env, releaseName)
			if err != nil {
				return fmt.Errorf("getting release links: %w", err)
			}

			output, err := links.FormatLinks(releaseLinks, linkName)
			if err != nil {
				return fmt.Errorf("formatting links: %w", err)
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (interactive if not specified)")

	return cmd
}

func NewReleaseSchemaCmd() *cobra.Command {
	var env string
	cmd := &cobra.Command{
		Use:  "schema",
		Args: cobra.RangeArgs(0, 1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 && env == "" {
				return fmt.Errorf("environment (--env flag) is required when querying a release")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), v1alpha1.ReleaseSpecification())
				return err
			}

			cat := catalog.FromContext(cmd.Context())

			foundRelease, err := cat.LookupRelease(env, args[0])
			if err != nil {
				return fmt.Errorf("looking up release: %w", err)
			}

			cfg := config.FromContext(cmd.Context())

			charts := helm.ChartCache{
				Refs:            cfg.Charts,
				DefaultChartRef: cfg.DefaultChartRef,
				Root:            cfg.JoyCache,
				Puller:          helm.CLI{IO: internal.IO{Out: cmd.ErrOrStderr(), Err: cmd.ErrOrStderr()}},
			}

			chart, err := charts.GetReleaseChartFS(cmd.Context(), foundRelease)
			if err != nil {
				return fmt.Errorf("getting release chart: %w", err)
			}

			schemaBytes, err := chart.ReadFile("values.cue")
			if err != nil {
				if os.IsNotExist(err) {
					_, err := io.WriteString(cmd.OutOrStdout(), v1alpha1.ReleaseSpecification())
					return err
				}
				return fmt.Errorf("reading schema: %w", err)
			}

			schema := cuecontext.New().CompileBytes(schemaBytes).LookupPath(cue.MakePath(cue.Def("values")))

			_, err = fmt.Fprintln(cmd.OutOrStdout(), internal.StringifySchema(schema))
			return err
		},
	}

	cmd.Flags().StringVar(&env, "env", "", "environment to find release from")
	return cmd
}
