package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/yml"
	joy "github.com/nestoca/joy/pkg"
	"github.com/nestoca/joy/pkg/catalog"
)

const sourceYAML = `apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: backoffice
spec:
  project: backoffice
  version: 0.2569.0
  values:
    frontend:
      gateway:
        httpRoutes:
          main:
            hostnames: !lock
              - office.staging.nesto.ca
    env:
      ENV: !lock staging
      PUBLIC_API_PATH: !lock https://office.staging.nesto.ca/api
`

const envYAML = `apiVersion: joy.nesto.ca/v1alpha1
kind: Environment
metadata:
  name: staging
spec: {}`

const projectYAML = `apiVersion: joy.nesto.ca/v1alpha1
kind: Project
metadata:
  name: backoffice`

func newCatalog(t *testing.T) (cat *catalog.Catalog) {
	t.Helper()
	dir := t.TempDir()
	fmt.Println(dir)
	envDir := filepath.Join(dir, "environments", "staging")
	relDir := filepath.Join(envDir, "releases", "origination")
	projDir := filepath.Join(dir, "projects")
	require.NoError(t, os.MkdirAll(relDir, 0o755))
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(relDir, "backoffice.yaml"), []byte(sourceYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(envDir, "env.yaml"), []byte(envYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projDir, "backoffice.yaml"), []byte(projectYAML), 0o644))

	cat, err := joy.LoadCatalog(t.Context(), dir)
	require.NoError(t, err)

	return cat
}

func previewPath(dir string) string {
	return filepath.Join(dir, "environments", "staging", "releases", "origination", "backoffice-og-1234.yaml")
}

func decode(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, yaml.Unmarshal(data, &m))
	return m
}

func TestCreate(t *testing.T) {
	cat := newCatalog(t)
	err := Create(CreateParams{
		Catalog: cat,
		Writer:  yml.DiskWriter,
		Env:     "staging",
		Release: "backoffice",
		Suffix:  "-og-1234",
		Version: "1.2.3-preview",
		Patches: []patch.Op{
			{Op: "remove", Path: "/spec/values/frontend/gateway"},
			{Op: "add", Path: "/spec/namespace", Value: "previews"},
			{Op: "add", Path: "/spec/values/image", Value: map[string]any{"name": "backoffice"}},
		},
		Replaces: []Replacement{
			{Search: "https://office.staging.", Replace: `https://__RELEASE____SUFFIX__.previews.staging.`},
		},
	})
	require.NoError(t, err)

	text, err := os.ReadFile(previewPath(cat.Dir))
	require.NoError(t, err)
	s := string(text)

	require.Contains(t, s, "name: backoffice-og-1234")
	require.Contains(t, s, "version: 1.2.3-preview")
	require.NotContains(t, s, "gateway")
	require.Contains(t, s, "PUBLIC_API_PATH: !lock https://backoffice-og-1234.previews.staging.nesto.ca/api")
	require.Contains(t, s, "ENV: !lock staging")

	m := decode(t, previewPath(cat.Dir))
	require.Equal(t, "true", m["metadata"].(map[string]any)["labels"].(map[string]any)[v1alpha1.PreviewLabel])
	require.Equal(t, "true", m["metadata"].(map[string]any)["annotations"].(map[string]any)[v1alpha1.PruneArgoAnnotation])
	spec := m["spec"].(map[string]any)
	require.Equal(t, "previews", spec["namespace"])
	require.Equal(t, "backoffice", spec["values"].(map[string]any)["image"].(map[string]any)["name"])
}

func TestDelete(t *testing.T) {
	cat := newCatalog(t)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0",
	}))
	require.FileExists(t, previewPath(cat.Dir))

	cat, err := joy.LoadCatalog(t.Context(), cat.Dir)
	require.NoError(t, err)

	require.NoError(t, Delete(DeleteParams{Catalog: cat, Env: "staging", Releases: []string{"backoffice-og-1234"}}))

	require.NoFileExists(t, previewPath(cat.Dir))

	// Deleting again is a no-op.
	require.NoError(t, Delete(DeleteParams{Catalog: cat, Env: "staging", Releases: []string{"backoffice-og-1234"}}))
}

func TestDeleteAll(t *testing.T) {
	cat := newCatalog(t)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0",
	}))
	require.FileExists(t, previewPath(cat.Dir))

	cat, err := joy.LoadCatalog(t.Context(), cat.Dir)
	require.NoError(t, err)

	require.NoError(t, Delete(DeleteParams{Catalog: cat, Env: "staging", All: true}))
	require.NoFileExists(t, previewPath(cat.Dir))
}

func TestCreateErrors(t *testing.T) {
	cat := newCatalog(t)
	base := CreateParams{Catalog: cat, Writer: yml.DiskWriter, Env: "staging", Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0"}

	unknown := base
	unknown.Release = "does-not-exist"
	require.Error(t, Create(unknown))

	noSuffix := base
	noSuffix.Suffix = ""
	require.Error(t, Create(noSuffix))

	noVersion := base
	noVersion.Version = ""
	require.Error(t, Create(noVersion))
}
