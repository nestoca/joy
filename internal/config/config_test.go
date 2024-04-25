package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValueMappingUnmarshalling(t *testing.T) {
	cases := []struct {
		Name     string
		Input    string
		Expected ValueMapping
	}{
		{
			Name:  "mapping only",
			Input: "{key: value}",
			Expected: ValueMapping{
				ReleaseIgnoreList: nil,
				Mappings:          map[string]any{"key": "value"},
			},
		},
		{
			Name:  "mapping and ignore list",
			Input: "{releaseIgnoreList: [test], mappings: {key: value}}",
			Expected: ValueMapping{
				ReleaseIgnoreList: []string{"test"},
				Mappings:          map[string]any{"key": "value"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var output ValueMapping
			require.NoError(t, yaml.Unmarshal([]byte(tc.Input), &output))
			require.Equal(t, tc.Expected, output, "%#v", output)
		})
	}
}
