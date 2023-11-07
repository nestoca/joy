package main

import "testing"

func TestRegexp(t *testing.T) {
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
