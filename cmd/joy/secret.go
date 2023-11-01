package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/secret"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sealed-secret",
		Aliases: []string{"sealed-secrets", "secrets", "secret", "sec"},
		Short:   "Manage sealed secrets",
		Long: `Manage sealed secrets, such as sealing (encrypting) secrets and importing public certificate from cluster.

This command requires the sealed-secrets kubeseal cli to be installed: https://github.com/bitnami-labs/sealed-secrets 
`,
	}
	cmd.AddCommand(NewSecretImportCmd())
	cmd.AddCommand(NewSecretSealCmd())
	return cmd
}

func NewSecretImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import sealed secrets public certificate from cluster",
		Long: `Import sealed secrets public certificate from cluster and store it into given environment CRD.

This command requires kubectl cli to be installed: https://kubernetes.io/docs/tasks/tools/#kubectl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return secret.ImportCert()
		},
	}
	return cmd
}

func NewSecretSealCmd() *cobra.Command {
	var env string
	cmd := &cobra.Command{
		Use:   "seal",
		Short: "Encrypt secret",
		Long: `Encrypt secret using public certificate of given environment's sealed secrets controller.

This command requires the sealed-secrets kubeseal cli to be installed: https://github.com/bitnami-labs/sealed-secrets
The sealed secrets public certificate must also have been imported into the environment using 'joy secret import' command.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load catalog
			loadOpts := catalog.LoadOpts{
				Dir:             cfg.CatalogDir,
				LoadEnvs:        true,
				SortEnvsByOrder: true,
			}
			cat, err := catalog.Load(loadOpts)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			return secret.Seal(cat, env)
		},
	}
	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment to seal secret in")
	return cmd
}
