package yml_test

import (
	"testing"

	"github.com/nestoca/joy/internal/yml"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		yaml1    string
		yaml2    string
		expected bool
	}{
		{
			name: "Identical docs",
			yaml1: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
    - item2
`,
			yaml2: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
    - item2
`,
			expected: true,
		},
		{
			name: "Different formatting and whitespace",
			yaml1: `
# Head comment
foo:
  bar: baz
  # Foot comment

  nested:
    - item1 # Line comment
    - item2
`,
			yaml2: `

# Head comment
foo:


    bar:  baz
    # Foot comment


    nested:  
      - item1    # Line comment
      - item2
`,
			expected: true,
		},
		{
			name: "Different values",
			yaml1: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
	- item2
`,
			yaml2: `
# Comment
foo:
  bar: bam
  nested:
    - item1 # Comment
	- item2
`,
			expected: false,
		},
		{
			name: "Different list items",
			yaml1: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
	- item2
`,
			yaml2: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
	- item3
`,
			expected: false,
		},
		{
			name: "Different comments",
			yaml1: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Comment
	- item2
`,
			yaml2: `
# Comment
foo:
  bar: baz
  nested:
    - item1 # Different comment
	- item2
`,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := yml.Compare([]byte(test.yaml1), []byte(test.yaml2))
			if actual != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, actual)
			}
		})
	}
}
