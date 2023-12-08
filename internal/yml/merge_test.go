package yml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type MergeCase struct {
	Name     string
	Src      string
	Dst      string
	Expected string
}

func TestYmlLocking(t *testing.T) {
	mergeTests := []MergeCase{
		{
			Name: "MergeLockedSubTreesIntoExistingSubTrees",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
			Dst: `
m: n
o: p
e:
  f: !lock q
  r: s
  j: !lock
    t: u
`,
			Expected: `
a: b
c: d
e:
  f: !lock q
  h: i
  j: !lock
    t: u
`,
		},
		{
			Name: "MergeMultipleComments",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
			Dst: `
m: n
o: p
e:
  # Normal comment before lock
  f: !lock q
  r: s
  # Normal comment after lock
  j: !lock
    t: u
`,
			Expected: `
a: b
c: d
e:
  # Normal comment before lock
  f: !lock q
  h: i
  # Normal comment after lock
  j: !lock
    t: u
`,
		},
		{
			Name: "MergeLineCommentLockMarker",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
			Dst: `
m: n
o: p
e:
  f: !lock q
  r: s
  j: !lock
    t: u
`,
			Expected: `
a: b
c: d
e:
  f: !lock q
  h: i
  j: !lock
    t: u
`,
		},
		{
			Name: "MergePreservingBraces",
			Src: `
a: {b: c}
`,
			Dst: `
a: b
d: !lock e
`,
			Expected: `
a: {b: c}
d: !lock e
`,
		},
		{
			Name: "MergeLockedSubTreesIntoNonExistingSubTrees",
			Src: `
a: b
c: d
`,
			Dst: `
m: n
o: p
e:
  f: !lock q
  r: s
  j: !lock
    t: u
`,
			Expected: `
a: b
c: d
`,
		},
		{
			Name: "MergeWhenAYAMLIsEmpty",
			Src:  `{}`,
			Dst: `
m: n
o: p
e:
  f: !lock q
  r: s
  j: !lock
    t: u
`,
			Expected: `{}`,
		},
		{
			Name: "MergeWhenBYAMLIsEmpty",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j:
    k: l
`,
			Dst: `{}`,
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
			Src:      `{}`,
			Dst:      `{}`,
			Expected: `{}`,
		},
		{
			Name:     "MergeWhenBothYAMLsAreNil",
			Expected: `{}`,
		},
		{
			Name: "MergeSanitizeLockedDestinationScalarsAsTodo",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j: !lock
    k: l
`,
			Dst: `
m: n
o: p
e:
  f: !lock q
  r: s
`,
			Expected: `
a: b
c: d
e:
  f: !lock q
  h: i
  j: !lock
    k: TODO
`,
		},
		{
			Name: "MergeSanitizeLockedDestinationScalarsAsTodoWhenTargetIsNil",
			Src: `
a: b
c: d
e:
  f: g
  h: i
  j: !lock
    k: l
`,
			Expected: `
a: b
c: d
e:
  f: g
  h: i
  j: !lock
    k: TODO
`,
		},
		{
			Name: "MergeArrays",
			Src: `
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
			Dst: `
m: n
o: p
e:
  f: !lock
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
  f: !lock
    - q
    - r
    - s
    - t
    - u
`,
		},
		{
			Name: "MergingLockedElementOnlyPresentInTargetAndNotLastOfHisSiblings_ShouldPreserveItsOrder",
			Src: `
a: b
c: d
e:
  f: g
  l: m
`,
			Dst: `
a: b
c: d
e:
  f: g
  h: i
  j: !lock k
  l: m
`,
			Expected: `
a: b
c: d
e:
  f: g
  l: m
  j: !lock k
`,
		},
		{
			Name: "MergingLockedElementWithinParentOnlyPresentInTargetAndFirstOfHisSiblings_ShouldPreserveItsOrder",
			Src: `
a: b
c:
  d: e
  i: j
`,
			Dst: `
a: b
c:
  f: 
    g: !lock h
  d: e
`,
			Expected: `
a: b
c:
  d: e
  i: j
`,
		},
		{
			Name: "MergingLockedElementWithinParentOnlyPresentInTargetAndNotLastOfHisSiblings_ShouldPreserveItsOrder",
			Src: `
a: b
c:
  d: e
  i: j
`,
			Dst: `
a: b
c:
  d: e
  f: 
    g: !lock h
  k: l
`,
			Expected: `
a: b
c:
  d: e
  i: j
`,
		},
		{
			Name: "MergingLockedElementToExistingUnlockedTargetElement_ShouldLockTargetElementAndLeaveItsValueUnchanged",
			Src: `
a: b
c:
  d: !lock e
  i: j
`,
			Dst: `
a: b
c:
  d: f
  i: j
`,
			Expected: `
a: b
c:
  d: f
  i: j
`,
		},
	}

	for _, test := range mergeTests {
		t.Run(test.Name, func(t *testing.T) { testMerge(t, test) })
	}
}

func testMerge(t *testing.T, testcase MergeCase) {
	// Parse a.yaml
	var src *yaml.Node
	if testcase.Src != "" {
		src = &yaml.Node{}
		if err := yaml.Unmarshal([]byte(testcase.Src), src); err != nil {
			t.Fatalf("Failed to parse a.yaml: %v", err)
		}
		src.Style = 0
	}

	// Parse b.yaml
	var dst *yaml.Node
	if testcase.Dst != "" {
		dst = &yaml.Node{}
		if err := yaml.Unmarshal([]byte(testcase.Dst), dst); err != nil {
			t.Fatalf("Failed to parse b.yaml: %v", err)
		}
		dst.Style = 0
	}

	// Merge
	result := Merge(dst, src)

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
	expected := strings.TrimSpace(testcase.Expected)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Mismatch (-expected +actual):%s", diff)
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
  j: !lock
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
  j: !lock
    k: TODO
`
	// Merge YAML nodes
	mergedNode := Merge(nil, &sourceNode)

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

func TestTodoMerge(t *testing.T) {
	testcases := []struct {
		Name     string
		Src      string
		Dst      string
		Expected string
	}{
		{
			Name:     "source updates empty dst",
			Src:      "{hello: world}",
			Dst:      "{}",
			Expected: "{hello: world}",
		},
		{
			Name:     "locked source updates empty dst",
			Src:      "{hello: !lock world}",
			Dst:      "{}",
			Expected: "{hello: !lock TODO}",
		},
		{
			Name:     "locked complex source updates empty",
			Src:      "{m: !lock {hello: world, maybe: [oui, non], people: {john: doe}}}",
			Dst:      "{}",
			Expected: "{m: !lock {hello: TODO, maybe: [TODO, TODO], people: {john: TODO}}}",
		},
		{
			Name:     "empty source removes destination keys",
			Src:      "{}",
			Dst:      "{hello: world}",
			Expected: "{}",
		},
		{
			Name:     "empty source does not affect locked dst keys",
			Src:      "{}",
			Dst:      "{hello: !lock world, john: doe}",
			Expected: "{hello: !lock world}",
		},
		{
			Name:     "lock source key does not affect lock destination key",
			Src:      "{hello: !lock no}",
			Dst:      "{hello: !lock yes}",
			Expected: "{hello: !lock yes}",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var src yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.Src), &src))

			var dst yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.Dst), &dst))

			result, err := yaml.Marshal(Merge(&dst, &src).Content[0])
			require.NoError(t, err)

			result = bytes.TrimSpace(result)

			require.Equal(t, tc.Expected, string(result))
		})
	}
}

func TestYmlMerge(t *testing.T) {
	cases := []struct {
		Name     string
		Src      string
		Dst      string
		Expected string
	}{
		{
			Name:     "conflicting types",
			Src:      "{key: 1}",
			Dst:      "{key: hello}",
			Expected: "{key: 1}",
		},
		{
			Name:     "conflicting types locked dst",
			Src:      "{key: 1}",
			Dst:      "{key: !lock hello}",
			Expected: "{key: !lock hello}",
		},
		{
			Name:     "conflicting types inner locked dst",
			Src:      "{key: 1}",
			Dst:      "{key: [!lock hello]}",
			Expected: "{key: 1}",
		},
		{
			Name:     "conflicting types locked src",
			Src:      "{key: !lock 1}",
			Dst:      "{key: hello}",
			Expected: "{key: hello}",
		},
		{
			Name:     "seq",
			Src:      "[1, 2, 3]",
			Dst:      "[]",
			Expected: "[1, 2, 3]",
		},
		{
			Name:     "locked seq",
			Src:      "!lock [1, 2, 3]",
			Dst:      "[]",
			Expected: "[]",
		},
		{
			Name:     "locked seq with dst values",
			Src:      "!lock [1, 2, 3]",
			Dst:      "[4, 5, 6]",
			Expected: "[4, 5, 6]",
		},
		{
			Name:     "seq with dst longer",
			Src:      "[1, 2, 3]",
			Dst:      "[4, 5, 6, 7, 8]",
			Expected: "[1, 2, 3]",
		},
		{
			Name:     "seq with dst longer and lock",
			Src:      "[1, 2, 3]",
			Dst:      "[4, 5, 6, 7, !lock 8]",
			Expected: "[1, 2, 3, !lock 8]",
		},
		{
			Name:     "seq with dst inner lock",
			Src:      "[1, 2, 3]",
			Dst:      "[4, !lock 5, 6, 7, !lock 8]",
			Expected: "[1, !lock 5, 3, !lock 8]",
		},
		{
			Name:     "seq with src inner lock",
			Src:      "[!lock 1, !lock 2, 3]",
			Dst:      "[      4, !lock 5, 6, 7, !lock 8]",
			Expected: "[4, !lock 5, 3, !lock 8]",
		},
		{
			Name:     "seq with locked dst",
			Src:      "[1, 2, 3]",
			Dst:      "!lock [4, 5, 6]",
			Expected: "!lock [4, 5, 6]",
		},
		{
			Name:     "map",
			Src:      "{key: 5}",
			Dst:      "{key: 3}",
			Expected: "{key: 5}",
		},
		{
			Name:     "map disjoint",
			Src:      "{key: 5}",
			Dst:      "{value: 3}",
			Expected: "{key: 5}",
		},
		{
			Name:     "map disjoint with dst lock",
			Src:      "{key: 5}",
			Dst:      "{value: !lock 3, foo: bar}",
			Expected: "{key: 5, value: !lock 3}",
		},
		{
			Name:     "map with special keys",
			Src:      "{key:1: a}",
			Dst:      "{}",
			Expected: "{'key:1': a}",
		},
		{
			Name:     "alias",
			Src:      "{a: &one 1, b: *one}",
			Dst:      "{a: &two 2, b: *two}",
			Expected: "{a: &one 1, b: *one}",
		},
		{
			Name:     "nested",
			Src:      "{a: {b: {c: d}}}",
			Dst:      "{a: {b: {c: e}}}",
			Expected: "{a: {b: {c: d}}}",
		},
		{
			Name:     "nested lock",
			Src:      "{a: {b: {c: d}}}",
			Dst:      "{a: {b: !lock {c: e}}}",
			Expected: "{a: {b: !lock {c: e}}}",
		},
		{
			Name: "real world",
			Src: `spec:
  containers:
    - name: mycontainer
      image: 2.0.0
`,
			Dst: `spec:
  containers:
    - name: mycontainer
      image: !lock 1.0.0
`,
			Expected: "spec:\n    containers:\n        - name: mycontainer\n          image: !lock 1.0.0",
		},
		{
			Name:     "styling discourage flow/do not use flow style if src does not",
			Src:      `- a`,
			Dst:      "[b]",
			Expected: "- a",
		},
		{
			Name:     "styling discourage flow/do not use flow style if dst does not",
			Src:      `[a]`,
			Dst:      "- b",
			Expected: "- a",
		},
		{
			Name: "styling discourage flow/do not use flow style if none use flow",
			Src:  `- a`,
			Dst:  "- b",

			Expected: "- a",
		},
		{
			Name:     "styling discourage flow/use flow if and only if both use flow",
			Src:      `[a]`,
			Dst:      "[b]",
			Expected: "[a]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var src, dst yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.Src), &src))
			require.NoError(t, yaml.Unmarshal([]byte(tc.Dst), &dst))

			actual, err := yaml.Marshal(Merge(&dst, &src))
			require.NoError(t, err)

			actual = bytes.TrimSpace(actual)

			require.Equal(t, tc.Expected, string(actual))
		})
	}
}
