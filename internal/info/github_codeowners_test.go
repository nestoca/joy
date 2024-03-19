package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCodeOwners(t *testing.T) {
	for _, tc := range []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "one-line",
			content:  "src/ @dj-martin @davidmdm @ayahajar @silphid",
			expected: []string{"dj-martin", "davidmdm", "ayahajar", "silphid"},
		},
		{
			name:     "multi-line",
			content:  "* @ayahajar\n\n\n* @dcpantalone\n",
			expected: []string{"ayahajar", "dcpantalone"},
		},
		{
			name:     "withcomments",
			content:  "# See https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners for syntax\n\n\n*   @dj-martin @davidmdm @ayahajar @silphid",
			expected: []string{"dj-martin", "davidmdm", "ayahajar", "silphid"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ElementsMatchf(t, parseCodeOwners(tc.content), tc.expected, "owners should match")
		})
	}
}
