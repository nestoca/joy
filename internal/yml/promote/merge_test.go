package promote_test

import (
	"bytes"
	"testing"

	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/internal/yml/promote"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYmlMerge(t *testing.T) {
	cases := []struct {
		Name     string
		Src      string
		Dst      string
		Joy      string
		Proposal string
	}{
		{
			Name:     "conflicting types",
			Src:      "{key: 1}",
			Dst:      "{key: hello}",
			Joy:      "{key: 1}",
			Proposal: "{key: 1}",
		},
		{
			Name:     "conflicting types locked dst",
			Src:      "{key: 1}",
			Dst:      "{key: !lock hello}",
			Joy:      "{key: !lock hello}",
			Proposal: "{key: !lock hello}",
		},
		{
			Name:     "conflicting types inner locked dst",
			Src:      "{key: 1}",
			Dst:      "{key: [!lock hello]}",
			Joy:      "{key: 1}",
			Proposal: "{key: 1}",
		},
		{
			Name:     "conflicting types locked src",
			Src:      "{key: !lock 1}",
			Dst:      "{key: hello}",
			Joy:      "{key: hello}",
			Proposal: "{key: hello}",
		},
		{
			Name:     "seq",
			Src:      "[1, 2, 3]",
			Dst:      "[]",
			Joy:      "[1, 2, 3]",
			Proposal: "[1, 2, 3]",
		},
		{
			Name:     "locked seq",
			Src:      "!lock [1, 2, 3]",
			Dst:      "[]",
			Joy:      "!lock [1, 2, 3]",
			Proposal: "!lock [!!int TODO, !!int TODO, !!int TODO]",
		},
		{
			Name:     "seq with dst longer",
			Src:      "[1, 2, 3]",
			Dst:      "[4, 5, 6, 7, 8]",
			Joy:      "[1, 2, 3]",
			Proposal: "[1, 2, 3]",
		},
		{
			Name:     "seq with dst longer and lock",
			Src:      "[1, 2, 3]",
			Dst:      "[4, 5, 6, 7, !lock 8]",
			Joy:      "[1, 2, 3]",
			Proposal: "[1, 2, 3, !lock 8]",
		},
		{
			Name:     "seq with dst inner lock",
			Src:      "[1, 2, 3]",
			Dst:      "[4, !lock 5, 6, 7, !lock 8]",
			Joy:      "[1, 2, 3]",
			Proposal: "[1, !lock 5, 3, !lock 8]",
		},
		{
			Name:     "seq with src inner lock",
			Src:      "[!lock 1, !lock 2, 3]",
			Dst:      "[      4, !lock 5, 6, 7, !lock 8]",
			Joy:      "[!lock TODO, !lock TODO, 3]",
			Proposal: "[4, !lock 5, 3, !lock 8]",
		},
		{
			Name:     "seq with locked dst",
			Src:      "[1, 2, 3]",
			Dst:      "!lock [4, 5, 6]",
			Joy:      "[1, 2, 3]",
			Proposal: "!lock [4, 5, 6]",
		},
		{
			Name:     "map",
			Src:      "{key: 5}",
			Dst:      "{key: 3}",
			Joy:      "{key: 5}",
			Proposal: "{key: 5}",
		},
		{
			Name:     "map disjoint",
			Src:      "{key: 5}",
			Dst:      "{value: 3}",
			Joy:      "{key: 5}",
			Proposal: "{key: 5}",
		},
		{
			Name:     "map disjoint with dst lock",
			Src:      "{key: 5}",
			Dst:      "{value: !lock 3, foo: bar}",
			Joy:      "{value: !lock 3, key: 5}",
			Proposal: "{key: 5, value: !lock 3}",
		},
		{
			Name:     "map with special keys",
			Src:      "{key:1: a}",
			Dst:      "{}",
			Joy:      "{'key:1': a}",
			Proposal: "{'key:1': a}",
		},
		{
			Name:     "alias",
			Src:      "{a: &one 1, b: *one}",
			Dst:      "{a: &two 2, b: *two}",
			Joy:      "{a: &one 1, b: *one}",
			Proposal: "{a: &one 1, b: *one}",
		},
		{
			Name:     "nested",
			Src:      "{a: {b: {c: d}}}",
			Dst:      "{a: {b: {c: e}}}",
			Joy:      "{a: {b: {c: d}}}",
			Proposal: "{a: {b: {c: d}}}",
		},
		{
			Name:     "nested lock",
			Src:      "{a: {b: {c: d}}}",
			Dst:      "{a: {b: !lock {c: e}}}",
			Joy:      "{a: {b: !lock {c: e}}}",
			Proposal: "{a: {b: !lock {c: e}}}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var src, dst yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.Src), &src))
			require.NoError(t, yaml.Unmarshal([]byte(tc.Dst), &dst))

			{
				actual, err := yaml.Marshal(yml.Merge(&src, &dst))
				require.NoError(t, err)

				actual = bytes.TrimSpace(actual)

				require.Equal(t, tc.Joy, string(actual), "JOY")
			}

			{
				actual, err := yaml.Marshal(promote.Merge(&dst, &src))
				require.NoError(t, err)

				actual = bytes.TrimSpace(actual)

				require.Equal(t, tc.Proposal, string(actual), "PROPOSAL")
			}
		})
	}
}
