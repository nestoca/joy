package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/formatting"
	"github.com/nestoca/joy/internal/project"
	"github.com/nestoca/joy/internal/release/list"
	"github.com/nestoca/joy/pkg/catalog"
)

func getCatalogDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "list"))
	require.NoError(t, err)
	return dir
}

func loadCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Load(context.Background(), getCatalogDir(t), nil)
	require.NoError(t, err)
	return cat
}

func sortedNonEmptyLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

func nonEmptyLinesInOrder(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func environmentNamesFromJSONInOrder(t *testing.T, raw string) []string {
	t.Helper()
	var envs []*v1alpha1.Environment
	require.NoError(t, json.Unmarshal([]byte(raw), &envs))
	names := make([]string, len(envs))
	for i, e := range envs {
		names[i] = e.Name
	}
	return names
}

func releaseNamesFromFlatJSON(t *testing.T, raw string) []string {
	t.Helper()
	var rels []*v1alpha1.Release
	require.NoError(t, json.Unmarshal([]byte(raw), &rels))
	names := make([]string, len(rels))
	for i, r := range rels {
		names[i] = r.Name
	}
	sort.Strings(names)
	return names
}

func releaseNamesByEnvFromJSON(t *testing.T, raw string) map[string][]string {
	t.Helper()
	var m map[string][]*v1alpha1.Release
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	out := make(map[string][]string, len(m))
	for env, rels := range m {
		names := make([]string, len(rels))
		for i, r := range rels {
			names[i] = r.Name
		}
		sort.Strings(names)
		out[env] = names
	}
	return out
}

func TestList_Environments(t *testing.T) {
	cat := loadCatalog(t)

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatJson))
		require.Equal(t, []string{"qa", "staging"}, environmentNamesFromJSONInOrder(t, buf.String()))
		var envs []*v1alpha1.Environment
		require.NoError(t, json.Unmarshal(buf.Bytes(), &envs))
		root := getCatalogDir(t)
		for _, e := range envs {
			wantRel := filepath.Join(e.Name, "env.yaml")
			require.Equal(t, wantRel, e.RelativePath, "env %s", e.Name)
			require.Equal(t, filepath.Join(root, wantRel), e.AbsolutePath, "env %s", e.Name)
		}
	})

	t.Run("yaml", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatYaml))
		out := buf.String()
		require.Contains(t, out, "name: qa")
		require.Contains(t, out, "name: staging")
		require.Contains(t, out, "kind: Environment")
		require.Contains(t, out, "relativePath: qa/env.yaml")
		require.Contains(t, out, "relativePath: staging/env.yaml")
		require.Contains(t, out, "absolutePath:")
	})

	t.Run("names", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatNames))
		require.Equal(t, []string{"qa", "staging"}, nonEmptyLinesInOrder(buf.String()))
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatTable))
		plain := stripansi.Strip(buf.String())
		require.Contains(t, plain, "NAME")
		require.Contains(t, plain, "OWNERS")
		require.Contains(t, plain, "qa")
		require.Contains(t, plain, "staging")
	})

	t.Run("rel-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatRelPaths))
		require.Equal(t, []string{"qa/env.yaml", "staging/env.yaml"}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("abs-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, environment.Render(cat, &buf, formatting.FormatAbsPaths))
		root := getCatalogDir(t)
		want := []string{
			filepath.Join(root, "qa", "env.yaml"),
			filepath.Join(root, "staging", "env.yaml"),
		}
		sort.Strings(want)
		require.Equal(t, want, sortedNonEmptyLines(buf.String()))
	})
}

func TestList_Projects(t *testing.T) {
	cat := loadCatalog(t)
	root := getCatalogDir(t)

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatJson))
		var projects []*v1alpha1.Project
		require.NoError(t, json.Unmarshal(buf.Bytes(), &projects))
		require.Len(t, projects, 3)
		sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
		for _, p := range projects {
			wantRel := filepath.Join("projects", p.Name+".yaml")
			require.Equal(t, wantRel, p.RelativePath, "project %s", p.Name)
			require.Equal(t, filepath.Join(root, wantRel), p.AbsolutePath, "project %s", p.Name)
		}
		for _, p := range cat.Projects {
			require.Empty(t, p.RelativePath, "catalog project %s must not be mutated by list output", p.Name)
			require.Empty(t, p.AbsolutePath, "catalog project %s must not be mutated by list output", p.Name)
		}
	})

	t.Run("yaml", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatYaml))
		out := buf.String()
		require.Contains(t, out, "name: service1")
		require.Contains(t, out, "name: service2")
		require.Contains(t, out, "name: service3")
		require.Contains(t, out, "kind: Project")
		require.Contains(t, out, "relativePath: projects/service1.yaml")
		require.Contains(t, out, "absolutePath:")
	})

	t.Run("names", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatNames))
		require.Equal(t, []string{"service1", "service2", "service3"}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatTable))
		plain := stripansi.Strip(buf.String())
		require.Contains(t, plain, "NAME")
		require.Contains(t, plain, "OWNERS")
		require.Contains(t, plain, "service1")
		require.Contains(t, plain, "service2")
		require.Contains(t, plain, "service3")
	})

	t.Run("rel-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatRelPaths))
		require.Equal(t, []string{
			"projects/service1.yaml",
			"projects/service2.yaml",
			"projects/service3.yaml",
		}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("abs-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, project.Render(cat, &buf, formatting.FormatAbsPaths))
		root := getCatalogDir(t)
		want := []string{
			filepath.Join(root, "projects", "service1.yaml"),
			filepath.Join(root, "projects", "service2.yaml"),
			filepath.Join(root, "projects", "service3.yaml"),
		}
		sort.Strings(want)
		require.Equal(t, want, sortedNonEmptyLines(buf.String()))
	})
}

func TestList_Releases_SingleEnvironmentFlatJSON(t *testing.T) {
	cat := loadCatalog(t)
	rl, err := list.GetReleaseList(cat, list.Params{
		Environments:         []string{"staging"},
		ReferenceEnvironment: "staging",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"staging"}, rl.Environments, "single selected env => flat JSON/YAML at top level")

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatJson, 0))
		trimmed := bytes.TrimSpace(buf.Bytes())
		require.Equal(t, byte('['), trimmed[0], "single environment: JSON should be a top-level array, not grouped by env")
		require.Equal(t, []string{"service1", "service3"}, releaseNamesFromFlatJSON(t, buf.String()))
		root := getCatalogDir(t)
		var rels []*v1alpha1.Release
		require.NoError(t, json.Unmarshal(trimmed, &rels))
		byName := make(map[string]*v1alpha1.Release, len(rels))
		for _, r := range rels {
			byName[r.Name] = r
		}
		require.Equal(t, filepath.Join("staging", "releases", "service1.yaml"), byName["service1"].RelativePath)
		require.Equal(t, filepath.Join(root, "staging", "releases", "service1.yaml"), byName["service1"].AbsolutePath)
		require.Equal(t, filepath.Join("staging", "releases", "service3.yaml"), byName["service3"].RelativePath)
		require.Equal(t, filepath.Join(root, "staging", "releases", "service3.yaml"), byName["service3"].AbsolutePath)
	})

	t.Run("yaml", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatYaml, 0))
		out := buf.String()
		require.Contains(t, out, "name: service1")
		require.Contains(t, out, "name: service3")
		require.Contains(t, out, "version: 1.2.3")
		require.Contains(t, out, "version: 3.4.5")
		require.Contains(t, out, "relativePath: staging/releases/service1.yaml")
		require.Contains(t, out, "absolutePath:")
	})

	t.Run("names", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatNames, 0))
		// Names list only cross-releases that have a release in the selected environment(s).
		require.Equal(t, []string{"service1", "service3"}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("rel-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatRelPaths, 0))
		require.Equal(t, []string{
			"staging/releases/service1.yaml",
			"staging/releases/service3.yaml",
		}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("abs-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatAbsPaths, 0))
		root := getCatalogDir(t)
		want := []string{
			filepath.Join(root, "staging", "releases", "service1.yaml"),
			filepath.Join(root, "staging", "releases", "service3.yaml"),
		}
		sort.Strings(want)
		require.Equal(t, want, sortedNonEmptyLines(buf.String()))
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatTable, 0))
		plain := stripansi.Strip(buf.String())
		require.Contains(t, plain, "NAME")
		require.Contains(t, plain, "STAGING")
		require.Contains(t, plain, "service1")
		require.Contains(t, plain, "1.2.3")
		require.Contains(t, plain, "service3")
		require.Contains(t, plain, "3.4.5")
		require.Contains(t, plain, "Reference Environment:")
	})
}

func TestList_Releases_MultipleEnvironmentsGroupedJSON(t *testing.T) {
	cat := loadCatalog(t)
	rl, err := list.GetReleaseList(cat, list.Params{
		Environments:         []string{"qa", "staging"},
		ReferenceEnvironment: "staging",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"qa", "staging"}, rl.Environments)

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatJson, 0))
		trimmed := bytes.TrimSpace(buf.Bytes())
		require.Equal(t, byte('{'), trimmed[0], "multiple environments: JSON should be a top-level object keyed by environment")
		byEnv := releaseNamesByEnvFromJSON(t, buf.String())
		require.Equal(t, []string{"service1", "service2"}, byEnv["qa"])
		require.Equal(t, []string{"service1", "service3"}, byEnv["staging"])
		root := getCatalogDir(t)
		var grouped map[string][]*v1alpha1.Release
		require.NoError(t, json.Unmarshal(trimmed, &grouped))
		find := func(env, name string) *v1alpha1.Release {
			for _, r := range grouped[env] {
				if r.Name == name {
					return r
				}
			}
			return nil
		}
		s1qa := find("qa", "service1")
		require.NotNil(t, s1qa)
		require.Equal(t, filepath.Join("qa", "releases", "service1.yaml"), s1qa.RelativePath)
		require.Equal(t, filepath.Join(root, "qa", "releases", "service1.yaml"), s1qa.AbsolutePath)
		s2qa := find("qa", "service2")
		require.NotNil(t, s2qa)
		require.Equal(t, filepath.Join("qa", "releases", "service2.yaml"), s2qa.RelativePath)
		s3st := find("staging", "service3")
		require.NotNil(t, s3st)
		require.Equal(t, filepath.Join("staging", "releases", "service3.yaml"), s3st.RelativePath)
		require.Equal(t, filepath.Join(root, "staging", "releases", "service3.yaml"), s3st.AbsolutePath)
	})

	t.Run("yaml", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatYaml, 0))
		out := buf.String()
		// Grouped output uses map keys; assert both trees are present.
		require.Contains(t, out, "qa:")
		require.Contains(t, out, "staging:")
		require.Contains(t, out, "name: service2")
		require.Contains(t, out, "name: service3")
		require.Contains(t, out, "relativePath: qa/releases/service2.yaml")
		require.Contains(t, out, "relativePath: staging/releases/service3.yaml")
		require.Contains(t, out, "absolutePath:")
	})

	t.Run("names", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatNames, 0))
		require.Equal(t, []string{"service1", "service2", "service3"}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("rel-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatRelPaths, 0))
		require.Equal(t, []string{
			"qa/releases/service1.yaml",
			"qa/releases/service2.yaml",
			"staging/releases/service1.yaml",
			"staging/releases/service3.yaml",
		}, sortedNonEmptyLines(buf.String()))
	})

	t.Run("abs-paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatAbsPaths, 0))
		root := getCatalogDir(t)
		want := []string{
			filepath.Join(root, "qa", "releases", "service1.yaml"),
			filepath.Join(root, "qa", "releases", "service2.yaml"),
			filepath.Join(root, "staging", "releases", "service1.yaml"),
			filepath.Join(root, "staging", "releases", "service3.yaml"),
		}
		sort.Strings(want)
		require.Equal(t, want, sortedNonEmptyLines(buf.String()))
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, list.Render(&buf, cat.Dir, rl, formatting.FormatTable, 0))
		plain := stripansi.Strip(buf.String())
		require.Contains(t, plain, "QA")
		require.Contains(t, plain, "STAGING")
		require.Contains(t, plain, "service1")
		require.Contains(t, plain, "Reference Environment:")
	})
}
