package rerender

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePullRequestRef(t *testing.T) {
	cases := []struct {
		ref            string
		expectedNumber int
		expectedRepo   string
		expectedError  bool
	}{
		{ref: "72789", expectedNumber: 72789},
		{ref: "#72789", expectedNumber: 72789},
		{ref: "https://github.com/nestoca/catalog/pull/72789", expectedNumber: 72789, expectedRepo: "nestoca/catalog"},
		{ref: "https://github.com/nestoca/catalog/pull/72789/files", expectedNumber: 72789, expectedRepo: "nestoca/catalog"},
		{ref: "https://github.com/nestoca/catalog/pull/72789#issuecomment-1", expectedNumber: 72789, expectedRepo: "nestoca/catalog"},
		{ref: "https://github.com/nestoca/catalog/issues/72789", expectedError: true},
		{ref: "github.com/nestoca/catalog/pull/72789", expectedError: true},
		{ref: "0", expectedError: true},
		{ref: "abc", expectedError: true},
	}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			number, repo, err := ParsePullRequestRef(c.ref)
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expectedNumber, number)
			require.Equal(t, c.expectedRepo, repo)
		})
	}
}
