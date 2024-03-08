package main

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/davidmdm/x/xerr"

	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/text"

	"github.com/nestoca/joy/internal/project"

	"github.com/TwiN/go-color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/jac"
	"github.com/nestoca/joy/internal/release"
	"github.com/nestoca/joy/internal/release/filtering"
	"github.com/nestoca/joy/internal/release/list"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/release/validate"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel", "r"},
		Short:   "Manage releases",
		Long:    `Manage releases, such as promoting a release in a given environment.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewReleaseListCmd())
	cmd.AddCommand(NewReleasePromoteCmd())
	cmd.AddCommand(NewReleaseSelectCmd())
	cmd.AddCommand(NewReleasePeopleCmd())
	cmd.AddCommand(NewReleaseRenderCmd())
	cmd.AddCommand(NewGitCommands())
	cmd.AddCommand(NewValidateCommand())

	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	var releases, envs, owners string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List releases across environments",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			// Filtering
			var filter filtering.Filter
			if releases != "" {
				filter = filtering.NewNamePatternFilter(releases)
			} else if len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			selectedEnvs := func() []string {
				if envs == "" {
					return cfg.Environments.Selected
				}
				return strings.Split(envs, ",")
			}()
			if owners != "" {
				filter = filtering.NewOwnerFilter(owners)
			}

			return list.List(list.Opts{
				CatalogDir:           cfg.CatalogDir,
				SelectedEnvs:         selectedEnvs,
				Filter:               filter,
				ReferenceEnvironment: cfg.ReferenceEnvironment,
			})
		},
	}
	cmd.Flags().StringVarP(&releases, "releases", "r", "", "Releases to list (comma-separated with wildcards, defaults to configured selection or all)")
	cmd.Flags().StringVarP(&envs, "env", "e", "", "environments to list (comma-separated, defaults to configured selection or all)")
	cmd.Flags().StringVarP(&owners, "owners", "o", "", "List releases by owners (comma-separated, defaults to all)")
	cmd.MarkFlagsMutuallyExclusive("releases", "owners")

	return cmd
}

func NewReleasePromoteCmd() *cobra.Command {
	var sourceEnv, targetEnv string
	var autoMerge, draft, dryRun, localOnly, noPrompt bool

	cmd := &cobra.Command{
		Use:     "promote [flags] [releases]",
		Aliases: []string{"prom", "p"},
		Short:   "Promote releases across environments",
		Args:    cobra.RangeArgs(0, 1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if autoMerge && draft {
				return fmt.Errorf("flags --auto-merge and --draft cannot be used together")
			}
			if noPrompt {
				if len(args) == 0 {
					return fmt.Errorf("releases are required when no-prompt is set")
				}
				if sourceEnv == "" {
					return fmt.Errorf("source environment is required when no-prompt is set")
				}
				if targetEnv == "" {
					return fmt.Errorf("target environment is required when no-prompt is set")
				}
			}

			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, releases []string) error {
			cfg := config.FromContext(cmd.Context())

			var filter filtering.Filter
			if len(releases) == 0 && len(cfg.Releases.Selected) > 0 {
				filter = filtering.NewSpecificReleasesFilter(cfg.Releases.Selected)
			}

			cat, err := catalog.Load(catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
				ReleaseFilter:   filter,
			})
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, sourceEnv)
			if err != nil {
				return err
			}

			targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, targetEnv)
			if err != nil {
				return err
			}

			selectedEnvironments := v1alpha1.GetEnvironmentsByNames(cat.Environments, cfg.Environments.Selected)

			opts := promote.Opts{
				Catalog:              cat,
				SourceEnv:            sourceEnv,
				TargetEnv:            targetEnv,
				Releases:             releases,
				ReleasesFiltered:     len(releases) > 0 || filter != nil,
				NoPrompt:             noPrompt,
				AutoMerge:            autoMerge,
				Draft:                draft,
				DryRun:               dryRun,
				LocalOnly:            localOnly,
				SelectedEnvironments: selectedEnvironments,
			}

			_, err = promote.NewDefaultPromotion(
				cfg.CatalogDir,
				cfg.GitHubOrganization,
				cfg.PromoteReleaseCommitTemplate,
				cfg.PromoteReleasePullRequestTemplate,
				cfg.RepositoriesDir,
				cfg.JoyCache,
				cfg.DefaultGitTagTemplate,
			).Promote(opts)
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

	return cmd
}

func NewReleaseSelectCmd() *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Select releases to include in listings and promotions",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return release.ConfigureSelection(cfg.CatalogDir, cfg.FilePath, allFlag)
		},
	}
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Select all releases (non-interactive)")
	return cmd
}

func NewReleasePeopleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owners",
		Short: "List people owning a release's project via jac cli",
		Long: `List people owning a release's project via jac cli.

Calls 'jac people --group <owner1>,<owner2>...' with the owners of the release's project.

All extra arguments and flags are passed to jac cli as is.

This command requires the jac cli: https://github.com/nestoca/jac
`,
		Aliases: []string{
			"owner",
			"own",
			"people",
		},
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return git.EnsureCleanAndUpToDateWorkingCopy(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return jac.ListReleasePeople(cfg.CatalogDir, args)
		},
	}
	return cmd
}

func NewReleaseRenderCmd() *cobra.Command {
	var (
		env          string
		colorEnabled bool
		gitRef       string
		diffRef      string
		diffContext  int
	)

	cmd := &cobra.Command{
		Use:   "render [release]",
		Short: "render kubernetes manifests from joy release",
		Args:  cobra.RangeArgs(0, 1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			var errs []error
			if diffRef != "" {
				if env == "" {
					errs = append(errs, fmt.Errorf("flag --env must be provided when --diff-ref is used"))
				}
				if len(args) == 0 {
					errs = append(errs, fmt.Errorf("release arg must be provided when --diff-ref is used"))
				}
			}
			return xerr.MultiErrFrom("validating flags and args", errs...)
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.FromContext(cmd.Context())

			var releaseName string
			if len(args) == 1 {
				releaseName = args[0]
			}

			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
			}

			buildRenderParams := func(buffer *bytes.Buffer) (render.RenderParams, error) {
				cat, err := catalog.Load(loadOpts)
				if err != nil {
					return render.RenderParams{}, fmt.Errorf("loading catalog: %w", err)
				}

				return render.RenderParams{
					Env:     env,
					Release: releaseName,
					Cache: helm.ChartCache{
						DefaultChart: cfg.DefaultChart,
						Root:         cfg.JoyCache,
						Puller:       helm.CLI{IO: internal.IoFromCommand(cmd)},
					},
					Catalog: cat,
					CommonRenderParams: render.CommonRenderParams{
						ValueMapping: cfg.ValueMapping,
						IO: internal.IO{
							Out: buffer,
							Err: cmd.ErrOrStderr(),
							In:  cmd.InOrStdin(),
						},
						Helm:  helm.CLI{},
						Color: colorEnabled,
					},
				}, nil
			}

			renderRef := func(ref string) (result string, err error) {
				if ref != "" {
					dirty, err := git.IsDirty(cfg.CatalogDir)
					if err != nil {
						return "", fmt.Errorf("checking if catalog is in dirty state: %w", err)
					}

					if dirty {
						if err := git.Stash(cfg.CatalogDir); err != nil {
							return "", fmt.Errorf("stashing catalog: %w", err)
						}
						defer func() {
							if applyErr := git.StashApply(cfg.CatalogDir); err == nil && applyErr != nil {
								err = fmt.Errorf("applying stash: %w", applyErr)
							}
						}()
					}

					if err := git.Checkout(cfg.CatalogDir, ref); err != nil {
						return "", fmt.Errorf("checking out: %s: %w", ref, err)
					}
					defer func() {
						if swichErr := git.SwitchBack(cfg.CatalogDir); err == nil && swichErr != nil {
							err = fmt.Errorf("switching git back to previous branch: %w", swichErr)
						}
					}()
				}

				var buf bytes.Buffer
				params, err := buildRenderParams(&buf)
				if err != nil {
					return "", err
				}

				if err := render.Render(cmd.Context(), params); err != nil {
					return "", err
				}

				return buf.String(), nil
			}

			gitRefResult, err := renderRef(gitRef)
			if err != nil {
				return err
			}

			if diffRef == "" {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), gitRefResult)
				return err
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

			diff := diffFunc(
				text.File{Name: cmp.Or(gitRef, "(current)"), Content: gitRefResult},
				text.File{Name: diffRef, Content: diffRefResult},
				diffContext,
			)

			_, err = fmt.Fprint(cmd.OutOrStdout(), diff)
			return err
		},
	}

	cmd.Flags().StringVar(&gitRef, "git-ref", "", "git ref to checkout before render")
	cmd.Flags().StringVar(&diffRef, "diff-ref", "", "git ref to checkout before render")
	cmd.Flags().IntVarP(&diffContext, "diff-context", "c", 8, "line context when rendering diff")

	cmd.Flags().StringVarP(&env, "env", "e", "", "environment to select release from.")
	cmd.Flags().BoolVar(&colorEnabled, "color", term.IsTerminal(int(os.Stdout.Fd())), "toggle output with color")
	return cmd
}

func NewValidateCommand() *cobra.Command {
	var env string

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

			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				EnvNames:        selectedEnvs,
				SortEnvsByOrder: true,
				ReleaseFilter:   releaseFilter,
			}

			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			var releases []*v1alpha1.Release
			for _, item := range cat.Releases.Items {
				for _, release := range item.Releases {
					if release == nil {
						continue
					}
					releases = append(releases, release)
				}
			}

			return validate.Validate(cmd.Context(), validate.ValidateParams{
				Releases:     releases,
				ValueMapping: cfg.ValueMapping,
				DefaultChart: cfg.DefaultChart,
				CacheRoot:    cfg.JoyCache,
				Helm: helm.CLI{
					IO: internal.IO{Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr(), In: cmd.InOrStdin()},
				},
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "environment to select release from.")
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

				if target == "" {
					target = cfg.ReferenceEnvironment
				}

				if target == "" {
					return fmt.Errorf("unable to determine target environment: specify target or set your reference environment in your config")
				}

				cat, err := catalog.Load(catalog.LoadOpts{Dir: cfg.CatalogDir})
				if err != nil {
					return err
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

				repoDir, err := project.GetCachedSourceDir(sourceRelease.Project, cfg.GitHubOrganization, cfg.RepositoriesDir, cfg.JoyCache)
				if err != nil {
					return fmt.Errorf("cloning repository: %w", err)
				}

				sourceTag, err := release.GetGitTag(sourceRelease, cfg.DefaultGitTagTemplate)
				if err != nil {
					return fmt.Errorf("getting tag for source version %s of release %s: %w", sourceRelease.Spec.Version, sourceRelease.Name, err)
				}

				targetTag, err := release.GetGitTag(targetRelease, cfg.DefaultGitTagTemplate)
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
