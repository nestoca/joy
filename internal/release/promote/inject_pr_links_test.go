package promote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectPullRequestLinks(t *testing.T) {
	template := `[#{{ .PullRequestNumber }}](https://github.com/{{ .Repository }}/pull/{{ .PullRequestNumber }})`
	repository := "acme/project"

	tests := []struct {
		name     string
		template string
		text     string
		expected string
	}{
		{
			name:     "empty",
			template: template,
			text:     "",
			expected: "",
		},
		{
			name:     "no pr number",
			template: template,
			text:     "no pr number",
			expected: "no pr number",
		},
		{
			name:     "just pr number",
			template: template,
			text:     "#123",
			expected: "[#123](https://github.com/acme/project/pull/123)",
		},
		{
			name:     "pr number in the middle",
			template: template,
			text:     "text #123 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text",
		},
		{
			name:     "multiple pr numbers on different lines",
			template: template,
			text:     "text #123 text\ntext #456 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text\n" +
				"text [#456](https://github.com/acme/project/pull/456) text",
		},
		{
			name:     "multiple pr numbers on the same line",
			template: template,
			text:     "text #123 text #456 text",
			expected: "text [#123](https://github.com/acme/project/pull/123) text [#456](https://github.com/acme/project/pull/456) text",
		},
		{
			name:     "non-pr numbers",
			template: template,
			text:     "text#123 text1#123 #123text",
			expected: "text#123 text1#123 #123text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := injectPullRequestLinks(repository, tt.text)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}
