package main

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmdm/x/xcontainer"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/k8s"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewClusterCmd() *cobra.Command {
	var (
		envName string
		env     *v1alpha1.Environment
		cluster k8s.ClusterDefinition
	)

	cmd := &cobra.Command{
		Use:  "cluster",
		Args: cobra.ArbitraryArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			cat := catalog.FromContext(cmd.Context())

			env, err = func() (*v1alpha1.Environment, error) {
				if envName == "" {
					return environment.SelectSingle(cat.Environments, nil, "select which environment")
				}
				env := environment.FindByName(cat.Environments, envName)
				if env == nil {
					return nil, fmt.Errorf("could not find env %q", envName)
				}
				return env, nil
			}()
			if err != nil {
				return err
			}

			data, err := os.ReadFile(filepath.Join(env.Dir, "cluster.yaml"))
			if err != nil {
				return fmt.Errorf("reading environment cluster definition file: %w", err)
			}

			if err := sigsyaml.Unmarshal(data, &cluster); err != nil {
				return fmt.Errorf("failed to parse cluster definition: %w", err)
			}

			if cluster.CRDs != "" {
				cluster.CRDs = filepath.Join(env.Dir, cluster.CRDs)
			}
			for _, source := range cluster.Sources {
				if source.Kubectl != nil && source.Kubectl.Path != "" && !strings.HasPrefix(source.Kubectl.Path, "https://") && !strings.HasPrefix(source.Kubectl.Path, "oci://") && !filepath.IsAbs(source.Kubectl.Path) {
					source.Kubectl.Path = filepath.Join(env.Dir, source.Kubectl.Path)
				}
			}
			cluster.Env = env
			cluster.Cat = cat
			cluster.Cat.WithEnvironments([]string{env.Name})
			cluster.KubeContext = cmp.Or(cluster.KubeContext, "orbstack")

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&envName, "env", "e", "", "environment name")

	cmd.AddCommand(NewClusterSetupCommand(&cluster))
	cmd.AddCommand(NewClusterDeployCommand(&cluster))
	cmd.AddCommand(NewClusterAuthCommand(&cluster))

	return cmd
}

func NewClusterSetupCommand(cluster *k8s.ClusterDefinition) *cobra.Command {
	var deploy bool

	cmd := &cobra.Command{
		Use:  "setup",
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := k8s.Setup(cmd.Context(), *cluster); err != nil {
				return fmt.Errorf("failed to setup cluster: %w", err)
			}
			if deploy {
				return Deploy(cmd.Context(), *cluster, true, nil)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&cluster.Recreate, "recreate", false, "recreate orbstack cluster")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "deploy environment resources")

	return cmd
}

func NewClusterAuthCommand(cluster *k8s.ClusterDefinition) *cobra.Command {
	var envName string
	cmd := &cobra.Command{
		Use: "auth",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cluster.Registries.Auth(cmd.Context(), cluster.KubeContext)
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "", "environment")

	return cmd
}

func NewClusterDeployCommand(cluster *k8s.ClusterDefinition) *cobra.Command {
	var envName string
	var watch bool

	cmd := &cobra.Command{
		Use: "deploy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Deploy(cmd.Context(), *cluster, watch, args)
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "environment", "setup cluster in orbstack defined by environment")
	cmd.Flags().BoolVar(&watch, "watch", false, "watch deployed resources for changes")

	return cmd
}

func Deploy(ctx context.Context, cluster k8s.ClusterDefinition, watch bool, projects []string) error {
	cat := catalog.FromContext(ctx)

	projectSet := func() xcontainer.Set[string] {
		if len(projects) == 0 {
			return nil
		}
		return xcontainer.ToSet(projects)
	}()

	filepaths, fileDirs := func() (xcontainer.Set[string], xcontainer.Set[string]) {
		paths := make(xcontainer.Set[string])
		dirs := make(xcontainer.Set[string])

		add := func(value string) {
			dirs.Add(filepath.Dir(value))
			value, _ = filepath.Rel(cat.Dir, value)
			paths.Add(value)
		}

		if cat.Root != nil {
			add(cat.Root.File.Path)
		}

		add(cluster.Env.File.Path)

		for _, project := range cat.Projects {
			if projectSet != nil && !projectSet.Has(project.Name) {
				continue
			}
			add(project.File.Path)
		}

		for _, cross := range cat.Releases.Items {
			for _, release := range cross.Releases {
				if release == nil {
					continue
				}
				if projectSet != nil && !projectSet.Has(release.Project.Name) {
					continue
				}
				add(release.File.Path)
			}
		}

		return paths, dirs
	}()

	fileChan, err := func() (chan string, error) {
		c := make(chan string, len(filepaths))
		for path := range filepaths {
			c <- path
		}

		if !watch {
			close(c)
			return c, nil
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}

		for dir := range fileDirs {
			watcher.Add(dir)
		}

		go func() {
			defer close(c)
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op == fsnotify.Chmod || event.Op == fsnotify.Rename {
						continue
					}
					if path, _ := filepath.Rel(cat.Dir, event.Name); filepaths.Has(path) {
						c <- path
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						watcher.Errors = nil
						continue
					}
					fmt.Fprintln(os.Stderr, fmt.Errorf("error from file watcher: %v", err))
				}
			}
		}()

		return c, nil
	}()
	if err != nil {
		return err
	}

	k8s.DeployResources(ctx, k8s.DeployResourcesParams{
		Cluster: cluster,
		Files:   fileChan,
	})

	return nil
}
