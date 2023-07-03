package release

import (
	"testing"
)

func TestGetIndentSize(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedSize int
	}{
		{
			name:         "Empty content",
			content:      "",
			expectedSize: 2,
		},
		{
			name: "No indentation",
			content: `name: John Doe
age: 30
`,
			expectedSize: 2,
		},
		{
			name: "Spaces indentation",
			content: `    name: John Doe
    age: 30
`,
			expectedSize: 4,
		},
		{
			name: "Spaces and hyphen indentation",
			content: `  - item1
  - item2
  - item3
`,
			expectedSize: 2,
		},
		{
			name: "Mixed spaces and empty Lines",
			content: `   
    name: John Doe
    age: 30
`,
			expectedSize: 4,
		},
		{
			name:         "Tab Indentation",
			content:      "\tname: John Doe\n\tage: 30\n",
			expectedSize: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size := getIndentSize(test.content)
			if size != test.expectedSize {
				t.Errorf("Expected indent size %d, but got %d for content:\n%s", test.expectedSize, size, test.content)
			}
		})
	}
}
