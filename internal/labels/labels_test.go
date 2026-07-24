package labels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSelectors(t *testing.T) {
	sels, err := ParseSelectors([]string{"bare", "k=v", "empty="})
	require.NoError(t, err)
	require.Equal(t, []Selector{
		{key: "bare"},
		{key: "k", value: "v", hasValue: true},
		{key: "empty", value: "", hasValue: true},
	}, sels)

	for _, bad := range []string{"", "=novalue", "   "} {
		_, err := ParseSelectors([]string{bad})
		require.Error(t, err, "selector %q should be rejected", bad)
	}
}

func TestSelectorMatches(t *testing.T) {
	// Bare key matches regardless of value (including empty), and misses when absent.
	require.True(t, Selector{key: "a"}.matches(map[string]string{"a": "anything"}))
	require.True(t, Selector{key: "a"}.matches(map[string]string{"a": ""}))
	require.False(t, Selector{key: "a"}.matches(map[string]string{"other": "x"}))
	require.False(t, Selector{key: "a"}.matches(nil))

	// key=value requires an exact match.
	require.True(t, Selector{key: "a", value: "1", hasValue: true}.matches(map[string]string{"a": "1"}))
	require.False(t, Selector{key: "a", value: "1", hasValue: true}.matches(map[string]string{"a": "2"}))
}

func TestFirstMatch(t *testing.T) {
	selectors, err := ParseSelectors([]string{"nesto.ca/preview", "team=fe"})
	require.NoError(t, err)

	got, ok := FirstMatch(selectors, map[string]string{"nesto.ca/preview": "true"})
	require.True(t, ok)
	require.Equal(t, "nesto.ca/preview", got.String())

	_, ok = FirstMatch(selectors, map[string]string{"unrelated": "x"})
	require.False(t, ok)
}
