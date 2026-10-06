package pr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsRequestableReviewer(t *testing.T) {
	cases := []struct {
		reviewer    string
		requestable bool
	}{
		{reviewer: "john.doe", requestable: true},
		{reviewer: "", requestable: false},
		{reviewer: "*", requestable: false},
		{reviewer: "github-actions[bot]", requestable: false},
		{reviewer: "renovate[bot]", requestable: false},
		{reviewer: "invalid-email-address", requestable: false},
		{reviewer: "nestobot", requestable: false},
		{reviewer: "NestoBot", requestable: false},
		{reviewer: "Copilot", requestable: false},
		{reviewer: "claude", requestable: false},
		{reviewer: "cursor", requestable: false},
	}

	for _, c := range cases {
		t.Run(c.reviewer, func(t *testing.T) {
			require.Equal(t, c.requestable, IsRequestableReviewer(c.reviewer))
		})
	}
}
