// Package preview creates, updates and deletes preview copies of a release in the catalog.
//
// A preview is a copy of a source release named <sourceRelease><suffix>, pinned to a specific
// build version and marked with the v1alpha1.PreviewLabel so that `joy build promote` excludes
// it. Callers layer their own transforms via patches (RFC 6902) and regex replacements.
package preview

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/davidmdm/x/xerr"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
	"gopkg.in/yaml.v3"
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
	Catalog  *catalog.Catalog
	Env      string
	Releases []string
	All      bool
}

// Create writes the preview copy of a release.
//
// New preview: copy source → built-ins (metadata.name, preview label, version) → patches →
// replacements → placeholder substitution (__RELEASE__, __SUFFIX__).
func Create(params CreateParams) error {
	source, err := params.Catalog.LookupRelease(params.Env, params.Release)
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

	text, err := func() ([]byte, error) {
		tree := yml.Clone(source.File.Tree)
		yml.SetOrAddNodeValue(tree, "metadata.name", target)
		yml.SetOrAddNodeValue(tree, "metadata.labels."+yml.EscapePathSegment(v1alpha1.PreviewLabel), "true")
		yml.SetOrAddNodeValue(tree, "metadata.annotations."+yml.EscapePathSegment(v1alpha1.PruneArgoAnnotation), "true")
		yml.SetOrAddNodeValue(tree, "spec.version", params.Version)

		if len(params.Patches) == 0 {
			return yaml.Marshal(tree)
		}

		patched, err := patch.Apply(tree, params.Patches)
		if err != nil {
			return nil, err
		}

		yml.CopyMetadata(patched, source.File.Tree)

		return yaml.Marshal(patched)
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
	file.Indent = source.File.Indent

	if err := params.Writer.WriteFile(file); err != nil {
		return fmt.Errorf("writing preview file: %w", err)
	}

	fmt.Printf("✅ Created preview %s at version %s\n", style.Resource(target), style.Version(params.Version))
	return nil
}

// Delete removes the preview copy of a release, if it exists.
func Delete(params DeleteParams) error {
	releases, err := func() (result []*v1alpha1.Release, err error) {
		if params.All {
			for _, cross := range params.Catalog.Releases.Items {
				for _, rel := range cross.Releases {
					if rel != nil && rel.Environment.Name == params.Env && rel.Labels[v1alpha1.PreviewLabel] == "true" {
						result = append(result, rel)
						break
					}
				}
			}
			return result, nil
		}

		var errs []error
		for _, name := range params.Releases {
			release, err := params.Catalog.LookupRelease(params.Env, name)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if release.Labels == nil || release.Labels[v1alpha1.PreviewLabel] != "true" {
				errs = append(errs, fmt.Errorf("%s/%s is not a preview", params.Env, name))
				continue
			}
			result = append(result, release)
		}
		return result, xerr.JoinOrdered(errs...)
	}()
	if err != nil {
		return fmt.Errorf("finding wanted preview releases: %w", err)
	}

	var errs []error
	for _, release := range releases {
		if err := os.Remove(release.File.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("%s/%s: %w", release.Environment.Name, release.Name, err))
		}
	}

	if err := xerr.JoinOrdered(errs...); err != nil {
		return fmt.Errorf("deleting previews: %w", err)
	}

	fmt.Printf("🗑️  Deleted %d preview(s)\n", len(releases))

	return nil
}
