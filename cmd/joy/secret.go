package main

import (
	"github.com/nestoca/joy/internal/secret"
	"github.com/spf13/cobra"
)

func NewSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "secret",
		Aliases: []string{"secrets", "sec"},
		Short:   "Manage sealed secrets",
		Long: `Manage sealed secrets, such as sealing (encrypting) secrets and importing public certificate from cluster.

This command requires the sealed-secrets kubeseal cli to be installed: https://github.com/bitnami-labs/sealed-secrets 
`,
		GroupID: "core",
	}
	cmd.AddCommand(NewSecretImportCmd())
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
