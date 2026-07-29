package yml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func nodeOf(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &n))
	return &n
}

func renderNode(t *testing.T, n *yaml.Node) string {
	t.Helper()
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(n))
	require.NoError(t, enc.Close())
	return b.String()
}

func TestCopyMetadata(t *testing.T) {
	// src: authored form — custom tag, a comment, and a deliberate (non-alphabetical) key order.
	src := nodeOf(t, `zebra: 1
spec:
  version: 9 # keep me
  env:
    LOCKED: !lock secret
`)
	// dst: same data but alphabetized, no tags/comments (as if rebuilt from JSON), plus a new key.
	dst := nodeOf(t, `spec:
  env:
    LOCKED: secret
    ADDED: new
  version: 9
zebra: 1
`)

	CopyMetadata(dst, src)
	out := renderNode(t, dst)

	// Custom tag restored.
	require.Contains(t, out, "LOCKED: !lock secret")
	// Comment restored.
	require.Contains(t, out, "keep me")
	// src key order restored (zebra before spec; version before env; LOCKED before the added key).
	require.Less(t, strings.Index(out, "zebra"), strings.Index(out, "spec"))
	require.Less(t, strings.Index(out, "version"), strings.Index(out, "env"))
	require.Less(t, strings.Index(out, "LOCKED"), strings.Index(out, "ADDED"))
	// dst-only key preserved (appended, not dropped).
	require.Contains(t, out, "ADDED: new")
}

func TestCopyMetadataDoesNotReconcileSequenceElements(t *testing.T) {
	// src: a !lock-tagged sequence whose first element also carries a custom tag.
	src := nodeOf(t, `list: !lock
  - !lock aaa
  - bbb
`)
	// dst: as if a patch removed/shifted elements (and the JSON round-trip dropped tags).
	dst := nodeOf(t, `list:
  - ccc
`)

	CopyMetadata(dst, src)

	list := dst.Content[0].Content[1] // document -> root mapping -> value of "list"
	// The sequence node's own tag is restored...
	require.Equal(t, "!lock", list.Tag)
	// ...but element metadata is NOT copied by index (no !lock leaks onto the wrong element).
	require.Len(t, list.Content, 1)
	require.NotEqual(t, "!lock", list.Content[0].Tag)
	require.Equal(t, "ccc", list.Content[0].Value)
}
