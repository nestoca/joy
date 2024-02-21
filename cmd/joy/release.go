package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/git/pr/github"
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
				return fmt.Errorf("no source release found")
			}
			if targetRelease == nil {
				return fmt.Errorf("no target release found")
			}

			sourceDir := cmp.Or(cfg.RepositoriesDir, filepath.Join(cfg.JoyCache, "src"))
			if err := os.MkdirAll(sourceDir, 0o755); err != nil {
				return fmt.Errorf("failed to ensure repoName cache: %w", err)
			}

			project := sourceRelease.Project

			repository := project.Spec.Repository
			if repository == "" {
				repository = fmt.Sprintf("%s/%s", cfg.GitHubOrganization, project.Name)
			}

			repoDir := filepath.Join(sourceDir, path.Base(repository))
			if _, err := os.Stat(repoDir); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}

				cloneOptions := github.CloneOptions{
					RepoURL: repository,
					OutDir:  repoDir,
				}
				if err := github.Clone(sourceDir, cloneOptions); err != nil {
					return fmt.Errorf("failed to clone project: %w", err)
				}
			}

			if err := git.FetchTags(repoDir); err != nil {
				return fmt.Errorf("fetching git tags: %w", err)
			}

			tmpl, err := func() (*template.Template, error) {
				templateSource := cmp.Or(project.Spec.GitTagTemplate, cfg.DefaultGitTagTemplate)
				if templateSource == "" {
					return nil, nil
				}
				return template.New("").Parse(templateSource)
			}()
			if err != nil {
				return fmt.Errorf("parsing config gitTagTemplate: %w", err)
			}

			getRevision := func(release *v1alpha1.Release) (string, error) {
				if tmpl == nil {
					return release.Spec.Version, nil
				}

				var buffer bytes.Buffer
				if err := tmpl.Execute(&buffer, struct{ Release *v1alpha1.Release }{release}); err != nil {
					return "", fmt.Errorf("executing template: %w", err)
				}

				return buffer.String(), nil
			}

			sourceTag, err := getRevision(sourceRelease)
			if err != nil {
				return fmt.Errorf("getting source tag from release: %w", err)
			}

			targetTag, err := getRevision(targetRelease)
			if err != nil {
				return fmt.Errorf("getting target tag from release: %w", err)
			}

			gitargs := append([]string{"git", command}, args[1:]...)
			gitargs = append(gitargs, sourceTag+".."+targetTag)

			fmt.Println(color.InCyan("running:"), color.InBold(strings.Join(gitargs, " ")))
			fmt.Println()

			gitCommand := exec.Command("git", gitargs[1:]...)
			gitCommand.Dir = repoDir
			gitCommand.Stdout = os.Stdout
			gitCommand.Stderr = os.Stderr
			gitCommand.Stdin = os.Stdin

			return gitCommand.Run()
		},
	}

	cmd.Flags().StringVar(&source, "from", "", "source environment to compare release from")
	cmd.Flags().StringVar(&target, "to", "", "target environment to compare release to")

	cmd.MarkFlagRequired("from")

	return cmd
}
