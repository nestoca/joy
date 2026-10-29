package main

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	cmd := &cobra.Command{
		Use:  "cluster",
		Args: cobra.ArbitraryArgs,
	}

	cmd.AddCommand(NewClusterSetupCommand())
	cmd.AddCommand(NewClusterDeployCommand())
	cmd.AddCommand(NewClusterAuthCommand())

	return cmd
}

func NewClusterSetupCommand() *cobra.Command {
	var (
		envName string
		cluster k8s.ClusterDefinition
		deploy  bool
	)

	cmd := &cobra.Command{
		Use: "setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())

			env, err := func() (*v1alpha1.Environment, error) {
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
				if source.Kubectl != nil && source.Kubectl.Path != "" && !filepath.IsAbs(source.Kubectl.Path) {
					source.Kubectl.Path = filepath.Join(env.Dir, source.Kubectl.Path)
				}
			}
			cluster.Env = env.Name
			cluster.KubeContext = cmp.Or(cluster.KubeContext, "orbstack")

			if err := k8s.Setup(cmd.Context(), cluster); err != nil {
				return fmt.Errorf("failed to setup cluster: %w", err)
			}

			if deploy {
				return Deploy(cmd.Context(), env, true)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "environment", "setup cluster in orbstack defined by environment")
	cmd.Flags().BoolVar(&cluster.Recreate, "recreate", false, "recreate orbstack cluster")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "deploy environment resources")

	return cmd
}

func NewClusterAuthCommand() *cobra.Command {
	var envName string
	cmd := &cobra.Command{
		Use: "auth",
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())

			env, err := func() (*v1alpha1.Environment, error) {
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

			var cluster k8s.ClusterDefinition
			if err := sigsyaml.Unmarshal(data, &cluster); err != nil {
				return fmt.Errorf("failed to parse cluster definition: %w", err)
			}

			cluster.Env = env.Name
			cluster.KubeContext = cmp.Or(cluster.KubeContext, "orbstack")

			return cluster.Registries.Auth(cmd.Context(), cluster.KubeContext)
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "", "environment")

	return cmd
}

func NewClusterDeployCommand() *cobra.Command {
	var envName string
	var watch bool

	cmd := &cobra.Command{
		Use: "deploy",
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())

			env, err := func() (*v1alpha1.Environment, error) {
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

			return Deploy(cmd.Context(), env, watch)
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "environment", "setup cluster in orbstack defined by environment")
	cmd.Flags().BoolVar(&watch, "watch", false, "watch deployed resources for changes")

	return cmd
}

func Deploy(ctx context.Context, env *v1alpha1.Environment, watch bool) error {
	cat := catalog.FromContext(ctx)

	cat.WithEnvironments([]string{env.Name})

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

		add(env.File.Path)

		for _, project := range cat.Projects {
			add(project.File.Path)
		}

		for _, cross := range cat.Releases.Items {
			for _, release := range cross.Releases {
				if release == nil {
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
		Namespace:   env.Name,
		RootDir:     cat.Dir,
		Files:       fileChan,
		KubeContext: "orbstack",
	})

	return nil
}
