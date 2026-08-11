package patch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const base = `spec:
  namespace: default
  values:
    env:
      ZULU: !lock z
      ALPHA: !lock a
    frontend:
      gateway:
        httpRoutes: {}
`

func doc(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var d yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &d))
	return &d
}

func mustOp(t *testing.T, s string) Op {
	t.Helper()
	op, err := ParseOp([]byte(s))
	require.NoError(t, err)
	return op
}

func render(t *testing.T, d *yaml.Node) string {
	t.Helper()
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(d))
	require.NoError(t, enc.Close())
	return b.String()
}

func TestApply(t *testing.T) {
	out, err := Apply(doc(t, base), []Op{
		mustOp(t, `{op: add, path: /spec/values/image, value: {name: backoffice}}`),
		mustOp(t, `{op: replace, path: /spec/values/env/ALPHA, value: A2}`),
		mustOp(t, `{op: remove, path: /spec/values/frontend/gateway}`),
	})
	require.NoError(t, err)
	s := render(t, out)

	// Custom tags restored by CopyMetadata (including on a replaced value).
	require.Contains(t, s, "ZULU: !lock z")
	require.Contains(t, s, "ALPHA: !lock A2")
	// Source key order preserved (ZULU before ALPHA), not alphabetized.
	require.Less(t, strings.Index(s, "ZULU"), strings.Index(s, "ALPHA"))
	// De-flowed to block style (no leftover JSON braces from the round-trip).
	require.NotContains(t, s, `{"`)
	// Patch effects.
	require.Contains(t, s, "name: backoffice")
	require.NotContains(t, s, "httpRoutes")
}

func TestApply_NoOpsReturnsInput(t *testing.T) {
	in := doc(t, base)
	out, err := Apply(in, nil)
	require.NoError(t, err)
	require.Same(t, in, out)
}

func TestApply_StrictErrors(t *testing.T) {
	// RFC 6902: replacing a non-existent path is an error.
	_, err := Apply(doc(t, base), []Op{mustOp(t, `{op: replace, path: /spec/values/nope, value: x}`)})
	require.Error(t, err)
	// add to a non-existent parent is an error.
	_, err = Apply(doc(t, base), []Op{mustOp(t, `{op: add, path: /spec/values/image/name, value: x}`)})
	require.Error(t, err)
}

func TestApplyMergePatch(t *testing.T) {
	out, err := ApplyMergePatch(doc(t, base), []byte(`
spec:
  values:
    image: {name: backoffice}
    env: {ALPHA: A2}
    frontend: {gateway: null}
`))
	require.NoError(t, err)
	s := render(t, out)

	// Custom tags restored by CopyMetadata (including on a replaced value).
	require.Contains(t, s, "ZULU: !lock z")
	require.Contains(t, s, "ALPHA: !lock A2")
	// Source key order preserved (ZULU before ALPHA), not alphabetized.
	require.Less(t, strings.Index(s, "ZULU"), strings.Index(s, "ALPHA"))
	// De-flowed to block style (no leftover JSON braces from the round-trip).
	require.NotContains(t, s, `{"`)
	// Merge effects: object merged in, null removes the key.
	require.Contains(t, s, "name: backoffice")
	require.NotContains(t, s, "httpRoutes")
	require.Contains(t, s, "namespace: default")
}

func TestApplyMergePatch_EmptyReturnsInput(t *testing.T) {
	in := doc(t, base)
	out, err := ApplyMergePatch(in, nil)
	require.NoError(t, err)
	require.Same(t, in, out)
}

func TestSetPathAndSetPathCreating(t *testing.T) {
	d := doc(t, base)
	root := d.Content[0]
	require.NoError(t, SetPath(root, []string{"spec", "namespace"}, Scalar("previews")))
	require.NoError(t, SetPathCreating(root, []string{"metadata", "labels", "joy.nesto.ca/preview"}, Scalar("true")))

	s := render(t, d)
	require.Contains(t, s, "namespace: previews")
	require.Contains(t, s, `joy.nesto.ca/preview: "true"`)
	// Existing !lock values untouched by the setters.
	require.Contains(t, s, "ZULU: !lock z")
}
