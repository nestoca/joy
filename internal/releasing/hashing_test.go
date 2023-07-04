package releasing

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestGetHash(t *testing.T) {
	tests := []struct {
		name     string
		yamlStr  string
		expected uint64
	}{
		{
			name: "Simple mapping",
			yamlStr: `
a: b
c: d
`,
			expected: 0x81022feed47920dd,
		},
		{
			name: "Nested mapping",
			yamlStr: `
a:
  b: c
  d: e
`,
			expected: 0xe3b9e596acbea44e,
		},
		{
			name: "Sequence",
			yamlStr: `
- a
- b
- c
`,
			expected: 0x4a2847aeffa76d18,
		},
		{
			name: "Locked subtree top level",
			yamlStr: `
a: b
## lock
c:
  d: e
`,
			expected: 0x902a9e6e46c3346a,
		},
		{
			name: "Locked subtree key value pair",
			yamlStr: `
a: b
c:
  ## lock
  d: e
`,
			expected: 0xa6eabfa4afad368f,
		},
		{
			name: "Complex",
			yamlStr: `
a: b
c:
  d: e
  f:
    - g
    - ## lock
      h: i
`,
			expected: 0xc7594a24449be1f6,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var doc yaml.Node
			err := yaml.Unmarshal([]byte(test.yamlStr), &doc)
			if err != nil {
				t.Fatalf("Error unmarshalling YAML: %v", err)
			}

			hash := GetHash(&doc)
			if hash != test.expected {
				t.Errorf("Expected hash code: %x, but got: %x", test.expected, hash)
			}
		})
	}
}

func TestHashForEquivalentDocs(t *testing.T) {
	tests := []struct {
		name     string
		yamlStr1 string
		yamlStr2 string
	}{
		{
			name: "Equivalent content",
			yamlStr1: `
a: b
c: d
`,
			yamlStr2: `
a: b

c:  d
`,
		},
		{
			name: "Different quoting, same values",
			yamlStr1: `
a: "b"
c: 'd'
`,
			yamlStr2: `
a: b
c: d
`,
		},
		{
			name: "Different comments, same structure",
			yamlStr1: `
# Header comment
a: b  # Line comment
# Footer comment

c: d  # Line comment
`,
			yamlStr2: `
a: b
c: d
`,
		},
		{
			name: "Different values in locked subtree, locked on mapping node key",
			yamlStr1: `
a: b
## lock
c:
  d: e
`,
			yamlStr2: `
a: b
## lock
f:
  g: h
`,
		},
		{
			name: "Different values in locked subtree, locked on key-value pair",
			yamlStr1: `
a: b
c:
  ## lock
  d: e
`,
			yamlStr2: `
a: b
c:
  ## lock
  f: g
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var doc1, doc2 yaml.Node
			err := yaml.Unmarshal([]byte(test.yamlStr1), &doc1)
			if err != nil {
				t.Fatalf("Error unmarshalling YAML: %v", err)
			}

			err = yaml.Unmarshal([]byte(test.yamlStr2), &doc2)
			if err != nil {
				t.Fatalf("Error unmarshalling YAML: %v", err)
			}

			hash1 := GetHash(&doc1)
			hash2 := GetHash(&doc2)

			if hash1 != hash2 {
				t.Errorf("Expected same hash codes for %s, but got different values.\nHash1: %x\nHash2: %x", test.name, hash1, hash2)
			}
		})
	}
}
