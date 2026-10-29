package schema

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestBuildTree(t *testing.T) {
	schema := cuecontext.New().CompileString(`
#values: {
	// Top-level API settings.
	api: {
		// CPU request for the API container.
		cpu?: string | *"100m"
		memory: string
	}
	tags: [...string]
}
`).LookupPath(cue.ParsePath("#values"))

	root, err := BuildTree(schema, "#values")
	require.NoError(t, err)
	require.Equal(t, "#values", root.Name)
	require.True(t, root.CanExpand())
	require.True(t, root.IsExpanded())
	require.Len(t, root.Children, 2)

	api := findChild(t, root, "api")
	require.Equal(t, "api", api.Path)
	require.Equal(t, KindStruct, api.Kind)
	require.False(t, api.Optional)
	require.Contains(t, api.Docs, "Top-level API settings")
	require.False(t, api.IsExpanded())

	api.Expand()
	require.Len(t, api.Children, 2)

	cpu := findChild(t, api, "cpu")
	require.Equal(t, "api.cpu", cpu.Path)
	require.True(t, cpu.Optional)
	require.Equal(t, `"100m"`, cpu.Default)
	require.Contains(t, cpu.Docs, "CPU request")
	require.Contains(t, cpu.Type, "string")

	memory := findChild(t, api, "memory")
	require.False(t, memory.Optional)
	require.Empty(t, memory.Default)
	require.Equal(t, "string", memory.Type)

	tags := findChild(t, root, "tags")
	require.Equal(t, KindList, tags.Kind)
	require.True(t, tags.CanExpand(), "open list [...string] should expose element pattern")
}

func TestBuildTreeMissing(t *testing.T) {
	value := cuecontext.New().CompileString(`foo: 1`).LookupPath(cue.ParsePath("missing"))
	_, err := BuildTree(value, "#values")
	require.Error(t, err)
}

func TestBuildTreeDisjunctionAndOptionalRef(t *testing.T) {
	schema := cuecontext.New().CompileString(`
#observability: {
	tier: int
	slo:  false | *{target: number}
}
#api: {
	replicas: int | *2
}
#values: {
	observability: false | *#observability
	api?:          #api
}
`).LookupPath(cue.ParsePath("#values"))

	root, err := BuildTree(schema, "#values")
	require.NoError(t, err)

	obs := findChild(t, root, "observability")
	require.True(t, obs.CanExpand(), "observability should expand into #observability fields")
	obs.Expand()
	require.True(t, len(obs.Children) >= 2)
	require.NotNil(t, findChild(t, obs, "tier"))
	require.NotNil(t, findChild(t, obs, "slo"))

	slo := findChild(t, obs, "slo")
	require.True(t, slo.CanExpand(), "slo disjunction should expand into struct fields")
	slo.Expand()
	require.NotNil(t, findChild(t, slo, "target"))

	api := findChild(t, root, "api")
	require.True(t, api.Optional)
	require.True(t, api.CanExpand(), "optional #api ref should expand")
	api.Expand()
	require.NotNil(t, findChild(t, api, "replicas"))
}

func TestBuildTreePatternMap(t *testing.T) {
	schema := cuecontext.New().CompileString(`
#scalar: number | string | bool
#values: {
	env: [string]: (#scalar | {secret: #scalar})
	tags: [...string]
}
`).LookupPath(cue.ParsePath("#values"))

	root, err := BuildTree(schema, "#values")
	require.NoError(t, err)

	env := findChild(t, root, "env")
	require.True(t, env.CanExpand(), "pattern map should expand")
	env.Expand()
	require.Len(t, env.Children, 1)

	entry := env.Children[0]
	require.Equal(t, "[string]", entry.Name)
	require.Equal(t, "env.[string]", entry.Path)
	require.True(t, entry.CanExpand(), "pattern value disjunction should reach struct branch")
	entry.Expand()
	require.NotNil(t, findChild(t, entry, "secret"))

	tags := findChild(t, root, "tags")
	require.True(t, tags.CanExpand(), "open list should expose element pattern")
	tags.Expand()
	require.NotEmpty(t, tags.Children)
	require.Contains(t, tags.Children[0].Type, "string")
}

func TestBuildTreeDisjunctionExposesAllStructBranches(t *testing.T) {
	schema := cuecontext.New().CompileString(`
#values: {
	// Disjunction whose default branch would otherwise hide the socket branch.
	bind: *{port: int} | {socket: string}
}
`).LookupPath(cue.ParsePath("#values"))

	root, err := BuildTree(schema, "#values")
	require.NoError(t, err)

	bind := findChild(t, root, "bind")
	require.True(t, bind.CanExpand())
	bind.Expand()
	require.NotNil(t, findChild(t, bind, "port"), "default branch field should be present")
	require.NotNil(t, findChild(t, bind, "socket"), "non-default branch field should not be hidden")
}

func findChild(t *testing.T, parent *Node, name string) *Node {
	t.Helper()
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}
	t.Fatalf("child %q not found under %q", name, parent.Name)
	return nil
}