package preview

import (
	"io"
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

// nestoPatches mirrors the built-in transforms the preview-service action passes.
func nestoPatches(t *testing.T) []patch.Op {
	t.Helper()
	specs := []string{
		`{op: remove, path: /spec/values/frontend/gateway}`,
		`{op: add, path: /spec/namespace, value: previews}`,
		`{op: add, path: /spec/values/image, value: {name: backoffice}}`,
	}
	ops := make([]patch.Op, 0, len(specs))
	for _, s := range specs {
		op, err := patch.ParseOp([]byte(s))
		require.NoError(t, err)
		ops = append(ops, op)
	}
	return ops
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
		Catalog: cat, Writer: yml.DiskWriter,
		Env: "staging", Release: "backoffice", Suffix: "-og-1234", Version: "1.2.3-preview",
		Patches: nestoPatches(t),
		Replaces: []Replacement{{
			Search:  `(https://)(office)(\.staging\..*/api)`,
			Replace: `${1}__RELEASE____SUFFIX__.previews$3`,
		}},
	})
	require.NoError(t, err)

	text, err := os.ReadFile(previewPath(dir))
	require.NoError(t, err)
	s := string(text)

	require.Contains(t, s, "name: backoffice-og-1234")
	require.Contains(t, s, "version: 1.2.3-preview")
	require.NotContains(t, s, "gateway")
	// placeholders resolved inside the regex replacement + !lock preserved:
	require.Contains(t, s, "PUBLIC_API_PATH: !lock https://backoffice-og-1234.previews.staging.nesto.ca/api")
	require.Contains(t, s, "ENV: !lock staging")

	m := decode(t, previewPath(dir))
	require.Equal(t, "true", m["metadata"].(map[string]any)["labels"].(map[string]any)[v1alpha1.PreviewLabel])
	spec := m["spec"].(map[string]any)
	require.Equal(t, "previews", spec["namespace"])
	require.Equal(t, "backoffice", spec["values"].(map[string]any)["image"].(map[string]any)["name"])
}

func TestCreateWithMergePatch(t *testing.T) {
	dir, cat := newCatalog(t)
	err := Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter,
		Env: "staging", Release: "backoffice", Suffix: "-og-1234", Version: "1.2.3-preview",
		MergePatch: []byte(`
metadata:
  annotations:
    argocd.nesto.ca/sync.prune: 'true'
spec:
  values:
    frontend:
      gateway: null
`),
		Replaces: []Replacement{{
			Search:  `(https://)(office)(\.staging\..*/api)`,
			Replace: `${1}__RELEASE____SUFFIX__.previews$3`,
		}},
	})
	require.NoError(t, err)

	text, err := os.ReadFile(previewPath(dir))
	require.NoError(t, err)
	s := string(text)

	require.Contains(t, s, "name: backoffice-og-1234")
	require.Contains(t, s, "version: 1.2.3-preview")
	require.NotContains(t, s, "gateway")
	require.Contains(t, s, "PUBLIC_API_PATH: !lock https://backoffice-og-1234.previews.staging.nesto.ca/api")

	// The slash in the annotation key needs no escaping with a merge patch.
	m := decode(t, previewPath(dir))
	annotations := m["metadata"].(map[string]any)["annotations"].(map[string]any)
	require.Equal(t, "true", annotations["argocd.nesto.ca/sync.prune"])
}

func TestCreateUpdateOnlyBumpsVersion(t *testing.T) {
	dir, cat := newCatalog(t)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0", Patches: nestoPatches(t),
	}))

	// Second create with a different version + a patch that must NOT be re-applied.
	extra, err := patch.ParseOp([]byte(`{op: add, path: /spec/values/SHOULD_NOT_APPEAR, value: x}`))
	require.NoError(t, err)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "9.9.9", Patches: []patch.Op{extra},
	}))

	text, err := os.ReadFile(previewPath(dir))
	require.NoError(t, err)
	require.Contains(t, string(text), "version: 9.9.9")
	require.NotContains(t, string(text), "SHOULD_NOT_APPEAR")
}

func TestCreateUpdateWarnsAboutSkippedPatches(t *testing.T) {
	_, cat := newCatalog(t)
	require.NoError(t, Create(CreateParams{
		Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
		Release: "backoffice", Suffix: "-og-1234", Version: "1.0.0", Patches: nestoPatches(t),
	}))

	captureStdout := func(fn func()) string {
		old := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		defer func() { os.Stdout = old }()

		fn()

		require.NoError(t, w.Close())
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		return string(out)
	}

	// Update without patches/replaces: no warning.
	out := captureStdout(func() {
		require.NoError(t, Create(CreateParams{
			Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
			Release: "backoffice", Suffix: "-og-1234", Version: "2.0.0",
		}))
	})
	require.NotContains(t, out, "not re-applied")

	// Update with patches: warns that they're skipped.
	out = captureStdout(func() {
		require.NoError(t, Create(CreateParams{
			Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
			Release: "backoffice", Suffix: "-og-1234", Version: "3.0.0", Patches: nestoPatches(t),
		}))
	})
	require.Contains(t, out, "backoffice-og-1234")
	require.Contains(t, out, "not re-applied")

	// Update with a merge patch (no --patch ops): also warns.
	out = captureStdout(func() {
		require.NoError(t, Create(CreateParams{
			Catalog: cat, Writer: yml.DiskWriter, Env: "staging",
			Release: "backoffice", Suffix: "-og-1234", Version: "4.0.0",
			MergePatch: []byte(`{spec: {values: {image: {name: backoffice}}}}`),
		}))
	})
	require.Contains(t, out, "not re-applied")
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
