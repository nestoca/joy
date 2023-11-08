package main

import "testing"

func TestLockMigratorRegexp(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "#lock",
			Output: "",
		},
		{
			Input:  "#lock\n# it!",
			Output: "# it!",
		},
		{
			Input:  "The quick brown fox jumps over the\n### lock \nlazy dog",
			Output: "The quick brown fox jumps over the\nlazy dog",
		},
		{
			Input:  "The quick brown fox jumps over the lazy dog\n#lock",
			Output: "The quick brown fox jumps over the lazy dog\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			actual := lockExpression.ReplaceAllString(tc.Input, "")
			if tc.Output != actual {
				t.Errorf("expected %q but got %q", tc.Output, actual)
			}
		})
	}
}
