package schema

import (
	"testing"
	"unicode/utf8"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func newTestModel(t *testing.T) browserModel {
	t.Helper()
	schema := cuecontext.New().CompileString(`
#values: {
	api: {
		cpu?: string | *"100m"
	}
}
`).LookupPath(cue.ParsePath("#values"))
	root, err := BuildTree(schema, "#values")
	require.NoError(t, err)
	return newBrowserModel(root)
}

// TestFilterBackspaceDeletesWholeRune guards against byte-wise deletion that
// would leave invalid UTF-8 in the filter for multi-byte characters.
func TestFilterBackspaceDeletesWholeRune(t *testing.T) {
	m := newTestModel(t)
	m.filtering = true

	model, _ := m.updateFilter(tea.KeyPressMsg{Code: 'é', Text: "é"})
	m = model.(browserModel)
	require.Equal(t, "é", m.filter)

	model, _ = m.updateFilter(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = model.(browserModel)
	require.Equal(t, "", m.filter)
	require.True(t, utf8.ValidString(m.filter), "filter must remain valid UTF-8 after backspace")
}
