package yml

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDeepCopyNode(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
	}{
		{
			name: "Simple",
			yamlData: `
foo: &anchor
  bar: baz
  nested:
    - item1
    - item2
`,
		},
		{
			name: "Nested",
			yamlData: `
foo:
  bar:
    baz: &anchor
      - item1
      - item2
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the YAML data into a Tree
			var node yaml.Node
			err := yaml.Unmarshal([]byte(test.yamlData), &node)
			if err != nil {
				t.Errorf("Failed to unmarshal YAML: %v", err)
				return
			}

			// Perform a deep copy of the node and its descendants
			clone := DeepCopyNode(&node)

			// Verify that the cloned nodes are not mere references
			verifyNoReference(&node, clone, t)
		})
	}
}

// Helper function to recursively verify that cloned nodes are not mere references
func verifyNoReference(original, clone *yaml.Node, t *testing.T) {
	// Verify the fields of the nodes are equal
	if !nodesEqual(original, clone) {
		t.Errorf("Cloned node does not match the original node: original=%+v, clone=%+v", original, clone)
		return
	}

	// Verify the content of the nodes
	if original.Content != nil {
		if len(original.Content) != len(clone.Content) {
			t.Errorf("Cloned node content length does not match the original node: original=%+v, clone=%+v", original, clone)
			return
		}

		for i := range original.Content {
			verifyNoReference(original.Content[i], clone.Content[i], t)
		}
	}
}

// Helper function to compare two nodes for equality
func nodesEqual(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || a.Anchor != b.Anchor || a.Alias != b.Alias {
		return false
	}

	return true
}
