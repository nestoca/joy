package patch

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const base = `
spec:
  namespace: default
  values:
    env:
      ENV: !lock staging
      PUBLIC_API_PATH: !lock https://office.staging.nesto.ca/api
    frontend:
      gateway:
        httpRoutes: {}
`

func root(t *testing.T, src string) (*yaml.Node, *yaml.Node) {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))
	return &doc, doc.Content[0]
}

func dump(t *testing.T, doc *yaml.Node) string {
	t.Helper()
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(doc))
	require.NoError(t, enc.Close())
	return b.String()
}

func mustOp(t *testing.T, spec string) Op {
	t.Helper()
	op, err := ParseOp([]byte(spec))
	require.NoError(t, err)
	return op
}

func TestApply_AddReplaceRemove(t *testing.T) {
	doc, r := root(t, base)
	ops := []Op{
		mustOp(t, `{op: add, path: /spec/values/image, value: {name: backoffice}}`),
		mustOp(t, `{op: replace, path: /spec/values/env/PUBLIC_API_PATH, value: https://backoffice-og-1234.previews.staging.nesto.ca/api}`),
		mustOp(t, `{op: remove, path: /spec/values/frontend/gateway}`),
	}
	require.NoError(t, Apply(r, ops))

	out := dump(t, doc)
	require.Contains(t, out, "name: backoffice")
	require.Contains(t, out, "PUBLIC_API_PATH: !lock https://backoffice-og-1234.previews.staging.nesto.ca/api",
		"replace must preserve the !lock tag")
	require.Contains(t, out, "ENV: !lock staging")
	require.NotContains(t, out, "gateway")
}

func TestApply_Errors(t *testing.T) {
	_, r := root(t, base)
	require.Error(t, Apply(r, []Op{mustOp(t, `{op: replace, path: /spec/values/env/NOPE, value: x}`)}))
	_, r = root(t, base)
	require.Error(t, Apply(r, []Op{mustOp(t, `{op: remove, path: /spec/values/nope}`)}))
	_, r = root(t, base)
	err := Apply(r, []Op{mustOp(t, `{op: add, path: /spec/values/image/name, value: x}`)})
	require.ErrorContains(t, err, "add the parent object instead")
	for _, op := range []string{"move", "copy", "test"} {
		_, r := root(t, base)
		require.Error(t, Apply(r, []Op{mustOp(t, "{op: "+op+", path: /spec/namespace, value: x}")}))
	}
}

func TestSetPathCreating(t *testing.T) {
	doc, r := root(t, base)
	require.NoError(t, SetPathCreating(r, []string{"metadata", "labels", "joy.nesto.ca/preview"}, Scalar("true")))
	require.NoError(t, SetPathCreating(r, []string{"spec", "values", "env", "NEW"}, Scalar("v")))

	out := dump(t, doc)
	require.Contains(t, out, `joy.nesto.ca/preview: "true"`)
	require.Contains(t, out, "NEW: v")
	require.Contains(t, out, "ENV: !lock staging", "existing siblings must survive")
}

func TestParseOp_NormalizesStyleToBlock(t *testing.T) {
	// JSON-authored values (flow map, double-quoted scalars) should render as clean block YAML.
	doc, r := root(t, base)
	op, err := ParseOp([]byte(`{"op":"add","path":"/spec/values/image","value":{"name":"backoffice"}}`))
	require.NoError(t, err)
	ns, err := ParseOp([]byte(`{"op":"add","path":"/spec/namespace","value":"previews"}`))
	require.NoError(t, err)
	require.NoError(t, Apply(r, []Op{op, ns}))

	out := dump(t, doc)
	require.Contains(t, out, "namespace: previews")  // not: namespace: "previews"
	require.Contains(t, out, "    name: backoffice") // block, not: image: {"name": ...}
	require.NotContains(t, out, `{"name"`)
}

func TestParsePointerEscapes(t *testing.T) {
	got, err := parsePointer("/a~1b/c~0d")
	require.NoError(t, err)
	require.Equal(t, []string{"a/b", "c~d"}, got)
	_, err = parsePointer("no-leading-slash")
	require.Error(t, err)
}
