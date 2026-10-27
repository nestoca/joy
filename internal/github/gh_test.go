package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsRepositoryNotFound(t *testing.T) {
	cases := []struct {
		Name     string
		Output   string
		Expected bool
	}{
		{
			Name:     "missing repository",
			Output:   "GraphQL: Could not resolve to a Repository with the name 'nestoca/m2m-tokens'. (repository)\n",
			Expected: true,
		},
		{
			Name:     "other failure",
			Output:   "error connecting to api.github.com\n",
			Expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, tc.Expected, isRepositoryNotFound(tc.Output))
		})
	}
}
