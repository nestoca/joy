package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepositoryFromPullRequestURL(t *testing.T) {
	cases := []struct {
		Name          string
		URL           string
		Expected      string
		ExpectedError string
	}{
		{
			Name:     "pull request url",
			URL:      "https://github.com/nestoca/catalog/pull/123",
			Expected: "nestoca/catalog",
		},
		{
			Name:          "not a pull request url",
			URL:           "https://github.com/nestoca/catalog/issues/123",
			ExpectedError: `unexpected pull request url "https://github.com/nestoca/catalog/issues/123"`,
		},
		{
			Name:          "other host",
			URL:           "https://example.com/nestoca/catalog/pull/123",
			ExpectedError: `unexpected pull request url "https://example.com/nestoca/catalog/pull/123"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			repository, err := repositoryFromPullRequestURL(tc.URL)
			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.Expected, repository)
		})
	}
}
