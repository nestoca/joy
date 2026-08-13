package yml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitIntoPathSegments(t *testing.T) {
	cases := []struct {
		Input    string
		Segments []string
	}{
		{
			Input:    "common",
			Segments: []string{"common"},
		},
		{
			Input:    "left.right",
			Segments: []string{"left", "right"},
		},
		{
			Input:    ".",
			Segments: []string{},
		},
		{Input: `hello.\..world`, Segments: []string{"hello", ".", "world"}},
		{
			Input:    `\.`,
			Segments: []string{"."},
		},
		{
			Input:    `left.mid\.dle.right`,
			Segments: []string{"left", "mid.dle", "right"},
		},
		{
			Input:    `hello\\world`,
			Segments: []string{`hello\world`},
		},
		{
			Input:    `hello\\\.world`,
			Segments: []string{`hello\.world`},
		},
		{
			Input:    `hello\\.world`,
			Segments: []string{`hello\`, `world`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			require.Equal(t, tc.Segments, SplitIntoPathSegments(tc.Input))
		})
	}
}
