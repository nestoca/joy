package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
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
	"github.com/nestoca/joy/pkg/catalog"
)

func NewReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases", "rel"},
		Short:   "Manage releases",
		Long:    `Manage releases, such as promoting a release in a given environment.`,
		GroupID: "core",
	}
	cmd.AddCommand(NewReleaseListCmd())
	cmd.AddCommand(NewReleasePromoteCmd())
	cmd.AddCommand(NewReleaseSelectCmd())
	cmd.AddCommand(NewReleasePeopleCmd())
	cmd.AddCommand(NewReleaseRenderCmd())
	cmd.AddCommand(NewGitQueryCommand("diff"))
	cmd.AddCommand(NewGitQueryCommand("log"))

	return cmd
}

func NewReleaseListCmd() *cobra.Command {
	var releases string
	var envs string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List releases across environments",
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

	return cmd
}

func NewReleasePromoteCmd() *cobra.Command {
	var releases string
	var sourceEnv, targetEnv string
	var autoMerge, draft bool

	cmd := &cobra.Command{
		Use:     "promote [flags] [releases]",
		Aliases: []string{"prom"},
		Short:   "Promote releases across environments",
		Args:    cobra.RangeArgs(0, 1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if autoMerge && draft {
				return fmt.Errorf("flags --auto-merge and --draft cannot be used together")
			}
			return nil
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

			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
				ReleaseFilter:   filter,
			}
			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			// Resolve source and target environments
			sourceEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, sourceEnv)
			if err != nil {
				return err
			}
			targetEnv, err := v1alpha1.GetEnvironmentByName(cat.Environments, targetEnv)
			if err != nil {
				return err
			}

			// Resolve environments selected by user via `joy env select`
			selectedEnvironments := v1alpha1.GetEnvironmentsByNames(cat.Environments, cfg.Environments.Selected)

			// Perform promotion
			opts := promote.Opts{
				Catalog:              cat,
				SourceEnv:            sourceEnv,
				TargetEnv:            targetEnv,
				ReleasesFiltered:     filter != nil,
				AutoMerge:            autoMerge,
				Draft:                draft,
				SelectedEnvironments: selectedEnvironments,
			}

			_, err = promote.NewDefaultPromotion(cfg.CatalogDir).Promote(opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&sourceEnv, "source", "s", "", "Source environment (interactive if not specified)")
	cmd.Flags().StringVarP(&targetEnv, "target", "t", "", "Target environment (interactive if not specified)")
	cmd.Flags().BoolVar(&autoMerge, "auto-merge", false, "Add auto-merge label to release PR")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create draft release PR")
	addArgumentsToUsage(cmd, "releases", "Comma-separated list of releases (interactive if not specified)")

	return cmd
}

// addArgumentsToUsage adds positional arguments and their descriptions to the usage template of a command.
func addArgumentsToUsage(cmd *cobra.Command, argumentsAndDescriptions ...string) {
	var builder strings.Builder
	builder.WriteString("Arguments:\n")
	for i := 0; i < len(argumentsAndDescriptions)-1; i += 2 {
		builder.WriteString(fmt.Sprintf("  %-21s %s\n", argumentsAndDescriptions[i], argumentsAndDescriptions[i+1]))
	}
	globalSectionPattern := regexp.MustCompile(`(?m)^Global Flags:`)
	cmd.SetUsageTemplate(globalSectionPattern.ReplaceAllString(cmd.UsageTemplate(), builder.String()+"\n$0"))
}

func NewReleaseSelectCmd() *cobra.Command {
	allFlag := false
	cmd := &cobra.Command{
		Use:     "select",
		Aliases: []string{"sel"},
		Short:   "Select releases to include in listings and promotions",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			return jac.ListReleasePeople(cfg.CatalogDir, args)
		},
	}
	return cmd
}

func NewReleaseRenderCmd() *cobra.Command {
	var (
		env   string
		color bool
	)

	cmd := &cobra.Command{
		Use:   "render [release]",
		Short: "render kubernetes manifests from joy release",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				SortEnvsByOrder: true,
			}

			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			var releaseName string
			if len(args) == 1 {
				releaseName = args[0]
			}

			io := internal.IO{
				Out: cmd.OutOrStdout(),
				Err: cmd.ErrOrStderr(),
				In:  cmd.InOrStdin(),
			}

			return render.Render(cmd.Context(), render.RenderOpts{
				Env:          env,
				Release:      releaseName,
				DefaultChart: cfg.DefaultChart,
				CacheDir:     cfg.JoyCache,
				ValueMapping: cfg.ValueMapping,
				Catalog:      cat,
				IO:           io,
				Helm:         helm.CLI{IO: io},
				Color:        color,
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "environment to select release from.")
	cmd.Flags().BoolVar(&color, "color", term.IsTerminal(int(os.Stdout.Fd())), "toggle output with color")
	return cmd
}

func NewGitQueryCommand(command string) *cobra.Command {
	var (
		source string
		target string
	)
	cmd := &cobra.Command{
		Use:  "git-" + command,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())

			cat, err := catalog.Load(catalog.LoadOpts{Dir: cfg.CatalogDir})
			if err != nil {
				return err
			}

			release := args[0]

			var sourceRelease, targetRelease *v1alpha1.Release

			for _, cross := range cat.Releases.Items {
				if cross.Name != release {
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
				return fmt.Errorf("no source found")
			}
			if targetRelease == nil {
				return fmt.Errorf("no target found")
			}

			repositoriesDir := cmp.Or(cfg.RepositoriesDir, filepath.Join(cfg.JoyCache, "src"))
			if err := os.MkdirAll(repositoriesDir, 0o755); err != nil {
				return fmt.Errorf("failed to ensure repoName cache: %w", err)
			}

			project := sourceRelease.Project

			repoName := project.Spec.Repository
			if repoName == "" {
				repoName = fmt.Sprintf("%s/%s", cfg.GitHubOrganization, project.Name)
			}

			repoDir := filepath.Join(repositoriesDir, strings.Split(repoName, "/")[1])
			if _, err := os.Stat(repoDir); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}

				clone := exec.Command("gh", "repo", "clone", repoName, repoDir)
				clone.Stdout = os.Stdout
				clone.Stderr = os.Stderr
				clone.Dir = repositoriesDir

				if err := clone.Run(); err != nil {
					return fmt.Errorf("failed to clone project: %w", err)
				}
			}

			getRevision := func(version string) string {
				if !strings.HasPrefix(version, "v") {
					version = "v" + version
				}
				if branch := semver.Prerelease(version); branch != "" {
					// Cannot work with nestoca -> our docker-tags are a subset of branch names and are modified
					// TODO: implement a heuristic? Can be faulty. Or do not support PRs?
					// For now the current implementation will try to match on the branch but will not find it.
					// which is just another way to fail.
					return branch
				}

				// TODO: discuss: this is a nesto sepcific assumption of the current situation. Perhaps some repositories won't
				// need a tag prefix. Think front-end as well
				subdirectory := cmp.Or(project.Spec.RepositorySubdirectory, "api")

				// TODO: discuss: version is preserving the `v` prefix from above. If a user doesn't use v prefixes this will
				// result in a useless git revision. Perhaps the project should provide a templated string instead:
				// git-tag-template: api/v{{ .Release.Spec.Version }}
				return path.Join(subdirectory, version)
			}

			fetch := exec.Command("git", "fetch", "--tags")
			fetch.Dir = repoDir

			if err := fetch.Run(); err != nil {
				return fmt.Errorf("failed to pull project: %w", err)
			}

			expr := getRevision(targetRelease.Spec.Version) + ".." + getRevision(sourceRelease.Spec.Version)

			gitargs := append([]string{"git", command}, args[1:]...)
			gitargs = append(gitargs, expr)

			fmt.Println(color.InCyan("running:"), color.InBold(strings.Join(gitargs, " ")))
			fmt.Println()

			gitCommand := exec.Command("git", gitargs[1:]...)
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

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("target")

	return cmd
}
