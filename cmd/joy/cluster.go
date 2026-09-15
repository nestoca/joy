package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/k8s"
	"github.com/nestoca/joy/pkg/catalog"
	"github.com/spf13/cobra"
	sigsyaml "sigs.k8s.io/yaml"
)

func NewClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "cluster",
		Args: cobra.ArbitraryArgs,
	}

	cmd.AddCommand(NewClusterSetupCommand())

	return cmd
}

func NewClusterSetupCommand() *cobra.Command {
	var (
		envName string
		cluster k8s.ClusterDefinition
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
			cluster.Env = env.Name

			return k8s.Setup(cmd.Context(), cluster)
		},
	}

	cmd.Flags().StringVarP(&envName, "env", "e", "environment", "setup cluster in orbstack defined by environment")
	cmd.Flags().BoolVar(&cluster.Recreate, "recreate", false, "recreate orbstack cluster")

	return cmd
}
