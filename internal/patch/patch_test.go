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
	out, err := Apply(
		doc(t, base),
		[]Op{
			{Op: "add", Path: "/spec/values/image", Value: map[string]any{"name": "backoffice"}},
			{Op: "replace", Path: "/spec/values/env/ALPHA", Value: "A2"},
			{Op: "remove", Path: "/spec/values/frontend/gateway"},
		},
	)
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
	_, err := Apply(doc(t, base), []Op{{Op: "replace", Path: "/spec/values/nope", Value: "x"}})
	require.Error(t, err)
	// add to a non-existent parent is an error.
	_, err = Apply(doc(t, base), []Op{{Op: "add", Path: "/spec/values/image/name", Value: "x"}})
	require.Error(t, err)
}
