package promotion

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestMergeLockedSubTreesIntoExistingSubTrees(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`
	bContent := `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`
	expectedContent := `
a: b
c: d
e:
  ## lock
  f: q
  h: i
  ## lock
  j:
    t: u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeMultipleComments(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`
	bContent := `
m: n
o: p
e:
  # Normal comment before lock
  ## lock
  f: q
  r: s
  ## lock
  # Normal comment after lock
  j:
    t: u
`
	expectedContent := `
a: b
c: d
e:
  # Normal comment before lock
  ## lock
  f: q
  h: i
  ## lock
  # Normal comment after lock
  j:
    t: u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeLineCommentLockMarker(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`
	bContent := `
m: n
o: p
e:
  f: q ##  lock
  r: s
  j: ##  lock
    t: u
`
	expectedContent := `
a: b
c: d
e:
  f: q ##  lock
  h: i
  j: ##  lock
    t: u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergePreservingBraces(t *testing.T) {
	aContent := `
a: {b: c}
`
	bContent := `
## lock
d: e
`
	expectedContent := `
a: {b: c}
## lock
d: e
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeLockedSubTreesIntoNonExistingSubTrees(t *testing.T) {
	aContent := `
a: b
c: d
`
	bContent := `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`
	expectedContent := `
a: b
c: d
e:
  ## lock
  f: q
  ## lock
  j:
    t: u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeWhenAYAMLIsEmpty(t *testing.T) {
	aContent := `{}`
	bContent := `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`
	expectedContent := `
e:
  ## lock
  f: q
  ## lock
  j:
    t: u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeWhenBYAMLIsEmpty(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`
	bContent := `{}`
	expectedContent := `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeWhenBothYAMLsAreEmpty(t *testing.T) {
	aContent := `{}`
	bContent := `{}`
	expectedContent := `{}`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeSanitizeLockedDestinationScalarsAsTodo(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j:
    k: l
`
	bContent := `
m: n
o: p
e:
  ## lock
  f: q
  r: s
`
	expectedContent := `
a: b
c: d
e:
  ## lock
  f: q
  h: i
  ## lock
  j:
    k: TODO
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func TestMergeArrays(t *testing.T) {
	aContent := `
a: b
c: d
e:
  f:
    - f
    - g
    - h
    - i
    - j
`
	bContent := `
m: n
o: p
e:
  ## lock
  f:
    - q
    - r
    - s
    - t
    - u
`
	expectedContent := `
a: b
c: d
e:
  ## lock
  f:
    - q
    - r
    - s
    - t
    - u
`

	testMergeYAMLFiles(t, aContent, bContent, expectedContent)
}

func testMergeYAMLFiles(t *testing.T, aContent, bContent, expectedContent string) {
	// Parse a.yaml
	var aMap yaml.Node
	if err := yaml.Unmarshal([]byte(aContent), &aMap); err != nil {
		t.Fatalf("Failed to parse a.yaml: %v", err)
	}
	aMap.Style = 0

	// Parse b.yaml
	var bMap yaml.Node
	if err := yaml.Unmarshal([]byte(bContent), &bMap); err != nil {
		t.Fatalf("Failed to parse b.yaml: %v", err)
	}
	bMap.Style = 0

	// Merge YAML nodes
	result := Merge(&aMap, &bMap)

	// Marshal the result with custom indentation
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(result); err != nil {
		t.Fatalf("Failed to marshal the result: %v", err)
	}

	// Get the encoded result
	cBytes := buf.Bytes()

	// Compare the result with the expected result
	actual := strings.TrimSpace(string(cBytes))
	expected := strings.TrimSpace(expectedContent)
	if actual != expected {
		t.Errorf("Merged YAML files do not match the expected result.\nActual:\n%s\nExpected:\n%s", actual, expected)
	}
}