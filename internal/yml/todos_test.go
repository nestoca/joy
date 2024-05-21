package yml_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

func TestHasLockedTodos(t *testing.T) {
	cases := []struct {
		Name     string
		Value    string
		Expected bool
	}{
		{
			Name: "no locked todos",
			Value: `{
				hello: bar,
				array: [1, 2, foo], 
				map: {
					john: doe,
				}
			}`,
			Expected: false,
		},
		{
			Name: "todos not locked",
			Value: `{
				hello: TODO,
				array: [TODO], 
				map: {
					john: TODO,
				}
			}`,
			Expected: false,
		},
		{
			Name:     "locked todo scalar",
			Value:    "!lock TODO",
			Expected: true,
		},
		{
			Name: "locked todo within map",
			Value: `{
				secret: {
					VALUE: !lock TODO
				}
			}`,
			Expected: true,
		},
		{
			Name: "todo within locked map",
			Value: `{
				secret: !lock {
					VALUE: TODO
				}
			}`,
			Expected: true,
		},
		{
			Name:     "locked todo deeply nested",
			Value:    "{top: !lock { middle: { bottom: TODO } } }",
			Expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.Value), &node))
			require.Equal(t, tc.Expected, yml.HasLockedTodos(&node))
		})
	}
}
