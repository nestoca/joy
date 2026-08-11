package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/preview"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

func NewReleasePreviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Create or delete preview copies of a release",
		Long: `Manage preview copies of a release within an environment.

A preview is a copy of a source release named <release><suffix>, pinned to a version and
labelled ` + "`joy.nesto.ca/preview`" + ` so that "joy build promote" excludes it. Callers layer
their own transforms via repeatable --patch (RFC 6902 op) flags, a single --merge-patch
(RFC 7386) flag, and repeatable --replace (regex) flags.`,
	}
	cmd.AddCommand(newReleasePreviewCreateCmd())
	cmd.AddCommand(newReleasePreviewDeleteCmd())
	return cmd
}

func newReleasePreviewCreateCmd() *cobra.Command {
	var (
		env        string
		suffix     string
		version    string
		patches    []string
		mergePatch string
		replaces   []string
	)

	cmd := &cobra.Command{
		Use:   "create -e <env> <release> --suffix <suffix> --version <version>",
		Short: "Create or update a preview copy of a release",
		Long: `Create a preview copy of <release> named <release><suffix> in the given environment.

Applies, in order: built-ins (metadata.name, joy.nesto.ca/preview label, spec.version), then
each --patch, then --merge-patch, then each --replace, then a final placeholder pass
(__RELEASE__, __SUFFIX__). If the preview already exists, only spec.version is updated.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ops, err := parsePatchFlags(patches)
			if err != nil {
				return err
			}
			replacements, err := parseReplaceFlags(replaces)
			if err != nil {
				return err
			}

			cat := catalog.FromContext(cmd.Context())
			cat.WithEnvironments([]string{env})

			return preview.Create(preview.CreateParams{
				Catalog:    cat,
				Writer:     yml.DiskWriter,
				Env:        env,
				Release:    args[0],
				Suffix:     suffix,
				Version:    version,
				Patches:    ops,
				MergePatch: []byte(mergePatch),
				Replaces:   replacements,
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment of the source release")
	cmd.Flags().StringVar(&suffix, "suffix", "", "Suffix appended to the release name to form the preview (include the leading dash, e.g. -og-1234)")
	cmd.Flags().StringVar(&version, "version", "", "Version to set on the preview release")
	cmd.Flags().StringArrayVar(&patches, "patch", nil, "(repeatable) A single RFC 6902 op (add/replace/remove) applied to the preview, as JSON/YAML")
	cmd.Flags().StringVar(&mergePatch, "merge-patch", "", "An RFC 7386 JSON Merge Patch applied to the preview, as JSON/YAML")
	cmd.Flags().StringArrayVar(&replaces, "replace", nil, "(repeatable) A single {search, replace} regex applied to the preview file text, as JSON/YAML")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("suffix")
	_ = cmd.MarkFlagRequired("version")

	return cmd
}

func newReleasePreviewDeleteCmd() *cobra.Command {
	var (
		env    string
		suffix string
	)

	cmd := &cobra.Command{
		Use:   "delete -e <env> <release> --suffix <suffix>",
		Short: "Delete a preview copy of a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cat := catalog.FromContext(cmd.Context())
			cat.WithEnvironments([]string{env})

			return preview.Delete(preview.DeleteParams{
				Catalog: cat,
				Env:     env,
				Release: args[0],
				Suffix:  suffix,
			})
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment of the release")
	cmd.Flags().StringVar(&suffix, "suffix", "", "Suffix identifying the preview (include the leading dash, e.g. -og-1234)")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("suffix")

	return cmd
}

func parsePatchFlags(specs []string) ([]patch.Op, error) {
	ops := make([]patch.Op, 0, len(specs))
	for _, spec := range specs {
		op, err := patch.ParseOp([]byte(spec))
		if err != nil {
			return nil, fmt.Errorf("parsing --patch %q: %w", spec, err)
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func parseReplaceFlags(specs []string) ([]preview.Replacement, error) {
	replacements := make([]preview.Replacement, 0, len(specs))
	for _, spec := range specs {
		var r preview.Replacement
		if err := yaml.Unmarshal([]byte(spec), &r); err != nil {
			return nil, fmt.Errorf("parsing --replace %q: %w", spec, err)
		}
		if r.Search == "" {
			return nil, fmt.Errorf("invalid --replace %q: search is empty", spec)
		}
		replacements = append(replacements, r)
	}
	return replacements, nil
}
