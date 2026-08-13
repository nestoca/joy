// Package preview creates, updates and deletes preview copies of a release in the catalog.
//
// A preview is a copy of a source release named <sourceRelease><suffix>, pinned to a specific
// build version and marked with the v1alpha1.PreviewLabel so that `joy build promote` excludes
// it. Callers layer their own transforms via patches (RFC 6902) and regex replacements.
package preview

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
	sigsyaml "sigs.k8s.io/yaml"
)

// Replacement is a single regex search/replace applied to the preview file text.
type Replacement struct {
	Search  string `yaml:"search" json:"search"`
	Replace string `yaml:"replace" json:"replace"`
}

// CreateParams are the inputs to Create.
type CreateParams struct {
	Catalog  *catalog.Catalog
	Writer   yml.Writer
	Env      string
	Release  string // source release name
	Suffix   string // appended to Release to form the preview name; includes any leading dash
	Version  string
	Patches  []patch.Op
	Replaces []Replacement
}

// DeleteParams are the inputs to Delete.
type DeleteParams struct {
	Catalog *catalog.Catalog
	Env     string
	Release string
	Suffix  string
}

// Create writes (or, if it already exists, version-bumps) the preview copy of a release.
//
// New preview: copy source → built-ins (metadata.name, preview label, version) → patches →
// replacements → placeholder substitution (__RELEASE__, __SUFFIX__). Existing preview: only
// spec.version is re-patched (copy is idempotent; other transforms are not re-applied).
func Create(params CreateParams) error {
	source, err := findSourceRelease(params.Catalog, params.Release, params.Env)
	if err != nil {
		return err
	}
	if params.Suffix == "" {
		return fmt.Errorf("suffix must not be empty")
	}
	if params.Version == "" {
		return fmt.Errorf("version must not be empty")
	}

	target := params.Release + params.Suffix
	targetPath := filepath.Join(filepath.Dir(source.File.Path), target+".yaml")

	source.Name = target
	source.Spec.Version = params.Version
	if source.Labels == nil {
		source.Labels = map[string]string{}
	}
	source.Labels[v1alpha1.PreviewLabel] = "true"

	text, err := func() ([]byte, error) {
		if len(params.Patches) == 0 {
			return sigsyaml.Marshal(source)
		}
		//
		// Caller patches (RFC 6902), applied via the JSON Patch library (tags/comments/order restored).
		patched, err := patch.Apply(source, params.Patches)
		if err != nil {
			return nil, err
		}

		patched.File, err = yml.NewFileFromObject(targetPath, source.File.Indent, patched)
		if err != nil {
			return nil, fmt.Errorf("building preview file: %w", err)
		}

		yml.CopyMetadata(patched.File.Tree, source.File.Tree)

		return patched.File.Yaml()
	}()
	if err != nil {
		return err
	}

	replacements := append(
		params.Replaces,
		[]Replacement{
			{Search: "__RELEASE__", Replace: params.Release},
			{Search: "__SUFFIX__", Replace: params.Suffix},
		}...,
	)

	for _, replacement := range replacements {
		text = bytes.ReplaceAll(text, []byte(replacement.Search), []byte(replacement.Replace))
	}

	file, err := yml.NewFile(targetPath, text)
	if err != nil {
		return fmt.Errorf("reconstructing patched file with text replacements: %w", err)
	}

	if err := params.Writer.WriteFile(file); err != nil {
		return fmt.Errorf("writing preview file: %w", err)
	}

	fmt.Printf("✅ Created preview %s at version %s\n", style.Resource(target), style.Version(params.Version))
	return nil
}

// Delete removes the preview copy of a release, if it exists.
func Delete(params DeleteParams) error {
	source, err := findSourceRelease(params.Catalog, params.Release, params.Env)
	if err != nil {
		return err
	}
	if params.Suffix == "" {
		return fmt.Errorf("suffix must not be empty")
	}

	target := params.Release + params.Suffix
	targetPath := filepath.Join(filepath.Dir(source.File.Path), target+".yaml")

	if _, err := os.Stat(targetPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("ℹ️ Preview %s does not exist; nothing to delete\n", style.Resource(target))
			return nil
		}
		return fmt.Errorf("checking preview file %s: %w", targetPath, err)
	}
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("removing preview file %s: %w", targetPath, err)
	}
	fmt.Printf("🗑️  Deleted preview %s\n", style.Resource(target))
	return nil
}

// findSourceRelease locates the source release within the (single-environment) catalog.
func findSourceRelease(cat *catalog.Catalog, name, env string) (*v1alpha1.Release, error) {
	for _, crossRelease := range cat.Releases.Items {
		if crossRelease.Name != name {
			continue
		}
		if len(crossRelease.Releases) == 0 || crossRelease.Releases[0] == nil || crossRelease.Releases[0].File == nil {
			return nil, fmt.Errorf("release %q not found in environment %q", name, env)
		}
		return crossRelease.Releases[0], nil
	}
	return nil, fmt.Errorf("release %q not found", name)
}
