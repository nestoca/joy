// Package preview creates, updates and deletes preview copies of a release in the catalog.
//
// A preview is a copy of a source release named <sourceRelease><suffix>, pinned to a specific
// build version and marked with the v1alpha1.PreviewLabel so that `joy build promote` excludes
// it. Callers layer their own transforms via patches (RFC 6902), a merge patch (RFC 7386) and
// regex replacements.
package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

// Replacement is a single regex search/replace applied to the preview file text.
type Replacement struct {
	Search  string `yaml:"search" json:"search"`
	Replace string `yaml:"replace" json:"replace"`
}

// CreateParams are the inputs to Create.
type CreateParams struct {
	Catalog    *catalog.Catalog
	Writer     yml.Writer
	Env        string
	Release    string // source release name
	Suffix     string // appended to Release to form the preview name; includes any leading dash
	Version    string
	Patches    []patch.Op
	MergePatch []byte // RFC 7386 JSON Merge Patch (YAML or JSON), applied after Patches
	Replaces   []Replacement
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
// merge patch → replacements → placeholder substitution (__RELEASE__, __SUFFIX__). Existing
// preview: only spec.version is re-patched (copy is idempotent; other transforms are not
// re-applied).
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

	if _, err := os.Stat(targetPath); err == nil {
		if len(params.Patches) > 0 || len(params.MergePatch) > 0 || len(params.Replaces) > 0 {
			fmt.Printf(
				"%s existing preview %s: only spec.version is updated, patches and replacements are not re-applied. Delete and recreate the preview to pick up changes to those.\n",
				style.Warning("⚠️"), style.Resource(target),
			)
		}
		return updateVersion(params.Writer, targetPath, params.Version, target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking preview file %s: %w", targetPath, err)
	}

	tree := yml.Clone(source.File.Tree)
	node := documentRoot(tree)

	// joy built-ins.
	if err := patch.SetPath(node, []string{"metadata", "name"}, patch.Scalar(target)); err != nil {
		return fmt.Errorf("setting metadata.name: %w", err)
	}
	if err := patch.SetPathCreating(node, []string{"metadata", "labels", v1alpha1.PreviewLabel}, patch.Scalar("true")); err != nil {
		return fmt.Errorf("setting preview label: %w", err)
	}
	if err := patch.SetPath(node, []string{"spec", "version"}, patch.Scalar(params.Version)); err != nil {
		return fmt.Errorf("setting spec.version: %w", err)
	}

	// Caller patches (RFC 6902), applied via the JSON Patch library (tags/comments/order restored).
	patched, err := patch.Apply(tree, params.Patches)
	if err != nil {
		return err
	}

	// Caller merge patch (RFC 7386), applied on top of the RFC 6902 patches.
	patched, err = patch.ApplyMergePatch(patched, params.MergePatch)
	if err != nil {
		return err
	}

	targetFile, err := yml.NewFileFromTree(targetPath, source.File.Indent, patched)
	if err != nil {
		return fmt.Errorf("building preview file: %w", err)
	}

	// Text transforms: regex replacements, then the final placeholder pass.
	text, err := applyReplacements(string(targetFile.Yaml), params.Replaces)
	if err != nil {
		return err
	}
	text = applyPlaceholders(text, params.Release, params.Suffix)
	targetFile.Yaml = []byte(text)

	if err := params.Writer.WriteFile(targetFile); err != nil {
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

func updateVersion(writer yml.Writer, path, version, target string) error {
	file, err := yml.LoadFile(path)
	if err != nil {
		return fmt.Errorf("loading preview file %s: %w", path, err)
	}
	if err := patch.SetPath(documentRoot(file.Tree), []string{"spec", "version"}, patch.Scalar(version)); err != nil {
		return fmt.Errorf("setting spec.version: %w", err)
	}
	if err := file.UpdateYamlFromTree(); err != nil {
		return fmt.Errorf("updating preview yaml: %w", err)
	}
	if err := writer.WriteFile(file); err != nil {
		return fmt.Errorf("writing preview file: %w", err)
	}
	fmt.Printf("✅ Updated preview %s to version %s\n", style.Resource(target), style.Version(version))
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

func applyReplacements(text string, replacements []Replacement) (string, error) {
	for _, r := range replacements {
		re, err := regexp.Compile(r.Search)
		if err != nil {
			return "", fmt.Errorf("invalid search regex %q: %w", r.Search, err)
		}
		text = re.ReplaceAllString(text, r.Replace)
	}
	return text, nil
}

func applyPlaceholders(text, release, suffix string) string {
	return strings.NewReplacer(
		"__RELEASE__", release,
		"__SUFFIX__", suffix,
	).Replace(text)
}

func documentRoot(tree *yaml.Node) *yaml.Node {
	if tree.Kind == yaml.DocumentNode && len(tree.Content) > 0 {
		return tree.Content[0]
	}
	return tree
}
