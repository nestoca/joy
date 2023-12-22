package yml_test

import (
	"testing"

	"gopkg.in/yaml.v3"

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

func TestEqualWithExclusion(t *testing.T) {
	tests := []struct {
		name         string
		yaml1        string
		yaml2        string
		excludedPath []string
		expected     bool
	}{
		{
			name: "Identical trees without exclusion",
			yaml1: `
foo:
  bar: baz
  nested:
    - item1
    - item2
`,
			yaml2: `
foo:
  bar: baz
  nested:
    - item1
    - item2
`,
			excludedPath: nil,
			expected:     true,
		},
		{
			name: "Different trees without exclusion",
			yaml1: `
foo:
  bar: baz
  nested:
    - item1
    - item2
`,
			yaml2: `
foo:
  bar: bam
  nested:
    - item1
    - item2
`,
			excludedPath: nil,
			expected:     false,
		},
		{
			name: "Identical trees excluding node at root level",
			yaml1: `
foo: baz
bar: qux
`,
			yaml2: `
foo: bam
bar: qux
`,
			excludedPath: []string{"foo"},
			expected:     true,
		},
		{
			name: "Identical trees excluding nested node",
			yaml1: `
foo:
  bar: baz
  nested:
    - item1
    - item2
`,
			yaml2: `
foo:
  bar: baz
  nested:
    - item3
    - item4
`,
			excludedPath: []string{"foo", "nested"},
			expected:     true,
		},
		{
			name: "Different trees excluding node at root level",
			yaml1: `
foo: baz
bar: qux
`,
			yaml2: `
foo: bam
bar: qul
`,
			excludedPath: []string{"foo"},
			expected:     false,
		},
		{
			name: "Different trees excluding nested node",
			yaml1: `
foo:
  bar: baz
  nested:
    - item1
    - item2
`,
			yaml2: `
foo:
  bar: bak
  nested:
    - item3
    - item4
`,
			excludedPath: []string{"foo", "nested"},
			expected:     false,
		},
		{
			name: "Different trees with non-matching exclusion",
			yaml1: `
foo:
  bar: baz
`,
			yaml2: `
foo:
  bar: bam
`,
			excludedPath: []string{"foo", "nested"},
			expected:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var a, b yaml.Node
			if err := yaml.Unmarshal([]byte(test.yaml1), &a); err != nil {
				t.Fatalf("error unmarshalling yaml1: %v", err)
			}
			if err := yaml.Unmarshal([]byte(test.yaml2), &b); err != nil {
				t.Fatalf("error unmarshalling yaml2: %v", err)
			}
			actual := yml.EqualWithExclusion(&a, &b, test.excludedPath...)
			if actual != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, actual)
			}
		})
	}
}
