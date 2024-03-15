package helm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestChartUnmarshalling(t *testing.T) {
	cases := []struct {
		Name     string
		Input    string
		Expected Chart
		Error    string
	}{
		{
			Name:  "raw string",
			Input: "oci://some.repo/path/to/chart:1.2.3",
			Expected: Chart{
				URL:     "oci://some.repo/path/to/chart",
				Version: "1.2.3",
			},
		},
		{
			Name:  "raw string no scheme",
			Input: "some.repo/path/to/chart:1.2.3",
			Expected: Chart{
				URL:     "oci://some.repo/path/to/chart",
				Version: "1.2.3",
			},
		},
		{
			Name:  "raw string no version",
			Input: "some.repo/path/to/chart",
			Error: "invalid chart: version required",
		},
		{
			Name:  "empty string",
			Input: `""`,
			Error: "invalid chart:\n  - url required\n  - version required",
		},
		{
			Name:  "structure",
			Input: "{ url: https://some.repo/path/to/chart, version: 1.2.3 }",
			Expected: Chart{
				URL:     "https://some.repo/path/to/chart",
				Version: "1.2.3",
			},
		},
		{
			Name:  "empty structure",
			Input: "{}",
			Error: "invalid chart:\n  - url required\n  - version required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var chart Chart
			err := yaml.Unmarshal([]byte(tc.Input), &chart)
			if tc.Error != "" {
				require.EqualError(t, err, tc.Error)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.Expected, chart)
		})
	}
}
