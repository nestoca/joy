package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
			return secret.ImportCert(catalog.FromContext(cmd.Context()))
		},
	}
	return cmd
}

func NewSecretSealCmd() *cobra.Command {
	var env string
	var noPrompt bool
	cmd := &cobra.Command{
		Use:   "seal",
		Short: "Encrypt secret",
		Example: `  # Base usage with prompts
  joy sealed-secret seal

  # With pre-selected environment
  joy sealed-secret seal -e staging

  # Seal secret content piped in from stdin (requires environment to be pre-selected)
  joy sealed-secret seal -e production < ./secret.txt`,
		Long: `Encrypt secret using public certificate of given environment's sealed secrets controller.

This command requires the sealed-secrets kubeseal cli to be installed: https://github.com/bitnami-labs/sealed-secrets
The sealed secrets public certificate must also have been imported into the environment using 'joy secret import' command.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())

			opts := secret.SealOptions{
				Env:         env,
				InputIsTTY:  term.IsTerminal(int(os.Stdin.Fd())),
				OutputIsTTY: term.IsTerminal(int(os.Stdout.Fd())),
				NoPrompt:    noPrompt,
			}

			if !opts.InputIsTTY && env == "" {
				return fmt.Errorf("environment must be provided via '--env' flag  when not using a tty")
			}

			return secret.Seal(cat, opts)
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment to seal secret in")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Run command without checks or prompts for input sanitization")

	return cmd
}
