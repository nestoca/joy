package yml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

type mergeTest struct {
	Name     string
	A        string
	B        string
	Expected string
}

var mergeTests = []mergeTest{
	{
		Name: "MergeLockedSubTreesIntoExistingSubTrees",
		A: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
		B: `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`,
		Expected: `
a: b
c: d
e:
  ## lock
  f: q
  h: i
  ## lock
  j:
    t: u
`,
	},
	{
		Name: "MergeMultipleComments",
		A: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
		B: `
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
`,
		Expected: `
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
`,
	},
	{
		Name: "MergeLineCommentLockMarker",
		A: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
		B: `
m: n
o: p
e:
  f: q ##  lock
  r: s
  j: ##  lock
    t: u
`,
		Expected: `
a: b
c: d
e:
  f: q ##  lock
  h: i
  j: ##  lock
    t: u
`,
	},
	{
		Name: "MergePreservingBraces",
		A: `
a: {b: c}
`,
		B: `
a: b
## lock
d: e
`,
		Expected: `
a: {b: c}
## lock
d: e
`,
	},
	{
		Name: "MergeLockedSubTreesIntoNonExistingSubTrees",
		A: `
a: b
c: d
`,
		B: `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`,
		Expected: `
a: b
c: d
e:
  ## lock
  f: q
  ## lock
  j:
    t: u
`,
	},
	{
		Name: "MergeWhenAYAMLIsEmpty",
		A:    `{}`,
		B: `
m: n
o: p
e:
  ## lock
  f: q
  r: s
  ## lock
  j:
    t: u
`,
		Expected: `
e:
  ## lock
  f: q
  ## lock
  j:
    t: u
`,
	},
	{
		Name: "MergeWhenBYAMLIsEmpty",
		A: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
		B: `{}`,
		Expected: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
	},
	{
		Name:     "MergeWhenBothYAMLsAreEmpty",
		A:        `{}`,
		B:        `{}`,
		Expected: `{}`,
	},
	{
		Name:     "MergeWhenBothYAMLsAreNil",
		Expected: `{}`,
	},
	{
		Name: "MergeSanitizeLockedDestinationScalarsAsTodo",
		A: `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j:
    k: l
`,
		B: `
m: n
o: p
e:
  ## lock
  f: q
  r: s
`,
		Expected: `
a: b
c: d
e:
  ## lock
  f: q
  h: i
  ## lock
  j:
    k: TODO
`,
	},
	{
		Name: "MergeSanitizeLockedDestinationScalarsAsTodoWhenTargetIsNil",
		A: `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j:
    k: l
`,
		Expected: `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j:
    k: TODO
`,
	},
	{
		Name: "MergeArrays",
		A: `
a: b
c: d
e:
  f:
    - f
    - g
    - h
    - i
    - j
`,
		B: `
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
`,
		Expected: `
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
`,
	},
	{
		Name: "MergingLockedElementOnlyPresentInTargetAndNotLastOfHisSiblings_ShouldPreserveItsOrder",
		A: `
a: b
c: d
e:
  f: g
  l: m
`,
		B: `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j: k
  l: m
`,
		Expected: `
a: b
c: d
e:
  f: g
  ## lock
  j: k
  l: m
`,
	},
	{
		Name: "MergingLockedElementWithinParentOnlyPresentInTargetAndFirstOfHisSiblings_ShouldPreserveItsOrder",
		A: `
a: b
c:
  d: e
  i: j
`,
		B: `
a: b
c:
  f: 
    ## lock
    g: h
  d: e
`,
		Expected: `
a: b
c:
  f:
    ## lock
    g: h
  d: e
  i: j
`,
	},
	{
		Name: "MergingLockedElementWithinParentOnlyPresentInTargetAndNotLastOfHisSiblings_ShouldPreserveItsOrder",
		A: `
a: b
c:
  d: e
  i: j
`,
		B: `
a: b
c:
  d: e
  f: 
    ## lock
    g: h
  k: l
`,
		Expected: `
a: b
c:
  d: e
  f:
    ## lock
    g: h
  i: j
`,
	},
	{
		Name: "MergingLockedElementToExistingUnlockedTargetElement_ShouldLockTargetElementAndLeaveItsValueUnchanged",
		A: `
a: b
c:
  ## lock
  d: e
  i: j
`,
		B: `
a: b
c:
  d: f
  i: j
`,
		Expected: `
a: b
c:
  ## lock
  d: f
  i: j
`,
	},
}

func TestMergeTableDriven(t *testing.T) {
	for _, test := range mergeTests {
		t.Run(test.Name, func(t *testing.T) {
			// Parse a.yaml
			var aMap *yaml.Node
			if test.A != "" {
				aMap = &yaml.Node{}
				if err := yaml.Unmarshal([]byte(test.A), aMap); err != nil {
					t.Fatalf("Failed to parse a.yaml: %v", err)
				}
				aMap.Style = 0
			}

			// Parse b.yaml
			var bMap *yaml.Node
			if test.B != "" {
				bMap = &yaml.Node{}
				if err := yaml.Unmarshal([]byte(test.B), bMap); err != nil {
					t.Fatalf("Failed to parse b.yaml: %v", err)
				}
				bMap.Style = 0
			}

			// Merge
			result := Merge(aMap, bMap)

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
			expected := strings.TrimSpace(test.Expected)
			if actual != expected {
				t.Errorf("Merged YAML files do not match the expected result.\nActual:\n%s\nExpected:\n%s", actual, expected)
			}
		})
	}
}

// Test that merge function copies everything from source to destination when destination is nil and that locked subtrees have their values replaced with TODO
func TestMergeWhenDestinationIsNil(t *testing.T) {
	// Source yaml
	source := `
a: b
c: d
e:	
  f: g
  h: i
  ## lock
  j:
    k: l
`
	var sourceNode yaml.Node
	if err := yaml.Unmarshal([]byte(source), &sourceNode); err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Expected result
	expected := `
a: b
c: d
e:
  f: g
  h: i
  ## lock
  j:
    k: TODO
`
	// Merge YAML nodes
	mergedNode := Merge(&sourceNode, nil)

	// Marshal the merged result
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(mergedNode); err != nil {
		t.Fatalf("Failed to marshal the result: %v", err)
	}
	mergedBytes := buf.Bytes()

	// Compare the result with the expected result
	actual := strings.TrimSpace(string(mergedBytes))
	expected = strings.TrimSpace(expected)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Mismatch (-expected +actual):%s", diff)
	}
}
