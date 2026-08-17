package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/patch"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
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

func newCatalog(t *testing.T) (dir string, cat *catalog.Catalog) {
	t.Helper()
	dir = t.TempDir()
	fmt.Println(dir)
	relDir := filepath.Join(dir, "environments", "staging", "releases", "origination")
	require.NoError(t, os.MkdirAll(relDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(relDir, "backoffice.yaml"), []byte(sourceYAML), 0o644))

	file, err := yml.LoadFile(filepath.Join(relDir, "backoffice.yaml"))
	require.NoError(t, err)
	rel, err := v1alpha1.LoadRelease(file)
	require.NoError(t, err)

	envs := []*v1alpha1.Environment{{EnvironmentMetadata: v1alpha1.EnvironmentMetadata{ObjectMeta: metav1.ObjectMeta{Name: "staging"}}}}
	cat = &catalog.Catalog{
		Environments: envs,
		Releases: cross.ReleaseList{
			Environments: envs,
			Items:        []*cross.Release{{Name: "backoffice", Releases: []*v1alpha1.Release{rel}}},
		},
	}
	return dir, cat
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
	dir, cat := newCatalog(t)
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

	text, err := os.ReadFile(previewPath(dir))
	require.NoError(t, err)
	s := string(text)

	require.Contains(t, s, "name: backoffice-og-1234")
	require.Contains(t, s, "version: 1.2.3-preview")
	require.NotContains(t, s, "gateway")
	require.Contains(t, s, "PUBLIC_API_PATH: !lock https://backoffice-og-1234.previews.staging.nesto.ca/api")
	require.Contains(t, s, "ENV: !lock staging")

	m := decode(t, previewPath(dir))
	require.Equal(t, "true", m["metadata"].(map[string]any)["labels"].(map[string]any)[v1alpha1.PreviewLabel])
	require.Equal(t, "true", m["metadata"].(map[string]any)["annotations"].(map[string]any)[v1alpha1.PruneArgoAnnotation])
	spec := m["spec"].(map[string]any)
	require.Equal(t, "previews", spec["namespace"])
	require.Equal(t, "backoffice", spec["values"].(map[string]any)["image"].(map[string]any)["name"])
}

func TestDelete(t *testing.T) {
	dir, cat := newCatalog(t)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0",
	}))
	require.FileExists(t, previewPath(dir))

	require.NoError(t, Delete(DeleteParams{Catalog: cat, Env: "staging", Release: "backoffice", Suffix: "-og-1234"}))
	require.NoFileExists(t, previewPath(dir))

	// Deleting again is a no-op.
	require.NoError(t, Delete(DeleteParams{Catalog: cat, Env: "staging", Release: "backoffice", Suffix: "-og-1234"}))
}

func TestCreateErrors(t *testing.T) {
	_, cat := newCatalog(t)
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
