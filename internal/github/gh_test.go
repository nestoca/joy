package github

import (
	"strings"
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

func TestTruncateBody(t *testing.T) {
	t.Run("body within limit is unchanged", func(t *testing.T) {
		body := strings.Repeat("a", maxBodyLength)
		require.Equal(t, body, truncateBody(body))
	})

	t.Run("body over limit is cut at a line break with a notice", func(t *testing.T) {
		line := strings.Repeat("é", 99) + "\n"
		body := strings.Repeat(line, 1000)

		truncated := truncateBody(body)

		require.LessOrEqual(t, len([]rune(truncated)), maxBodyLength)
		require.True(t, strings.HasSuffix(truncated, truncatedBodyNotice))
		require.True(t, strings.HasPrefix(body, strings.TrimSuffix(truncated, truncatedBodyNotice)+"\n"))
	})
}
