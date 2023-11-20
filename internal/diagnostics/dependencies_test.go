package diagnostics

import (
	"testing"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	requiredBy := []string{"cmd1", "cmd2"}
	bash := &dependencies.Dependency{
		Command:    "bash",
		Url:        "https://example.com/bash",
		RequiredBy: requiredBy,
	}
	sh := &dependencies.Dependency{
		Command:    "sh",
		Url:        "https://example.com/sh",
		RequiredBy: requiredBy,
	}
	missing := &dependencies.Dependency{
		Command:    "missing-dependency",
		Url:        "https://example.com/missing",
		RequiredBy: requiredBy,
	}

	testCases := []struct {
		Name     string
		Required []*dependencies.Dependency
		Optional []*dependencies.Dependency
		Expected Group
	}{
		{
			Name:     "no dependencies",
			Required: nil,
			Optional: nil,
			Expected: Group{
				Title:    "Dependencies",
				toplevel: true,
				SubGroups: Groups{
					{Title: "Required dependencies"},
					{Title: "Optional dependencies"},
				},
			},
		},
		{
			Name:     "all required and optional dependencies installed",
			Required: []*dependencies.Dependency{bash, sh},
			Optional: []*dependencies.Dependency{bash, sh},
			Expected: Group{
				Title:    "Dependencies",
				toplevel: true,
				SubGroups: Groups{
					{
						Title: "Required dependencies",
						Messages: Messages{
							{Type: success, Value: "bash installed"},
							{Type: success, Value: "sh installed"},
						},
					},
					{
						Title: "Optional dependencies",
						Messages: Messages{
							{Type: success, Value: "bash installed"},
							{Type: success, Value: "sh installed"},
						},
					},
				},
			},
		},
		{
			Name:     "missing required dependency",
			Required: []*dependencies.Dependency{bash, missing},
			Optional: []*dependencies.Dependency{bash, sh},
			Expected: Group{
				Title:    "Dependencies",
				toplevel: true,
				SubGroups: Groups{
					{
						Title: "Required dependencies",
						Messages: Messages{
							{Type: "success", Value: "bash installed"},
							{Type: "failed", Value: "missing-dependency missing (see https://example.com/missing)"},
						},
					},
					{
						Title: "Optional dependencies",
						Messages: Messages{
							{Type: "success", Value: "bash installed"},
							{Type: "success", Value: "sh installed"},
						},
					},
				},
			},
		},
		{
			Name:     "missing optional dependency",
			Required: []*dependencies.Dependency{bash, sh},
			Optional: []*dependencies.Dependency{bash, missing},
			Expected: Group{
				Title:    "Dependencies",
				toplevel: true,
				SubGroups: Groups{
					{
						Title: "Required dependencies",
						Messages: Messages{
							{Type: "success", Value: "bash installed"},
							{Type: "success", Value: "sh installed"},
						},
					},
					{
						Title: "Optional dependencies",
						Messages: Messages{
							{Type: "success", Value: "bash installed"},
							{
								Type: "failed", Value: "missing-dependency missing (see https://example.com/missing) but only required by:",
								Details: Messages{
									{Type: "info", Value: "joy cmd1"},
									{Type: "info", Value: "joy cmd2"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := diagnoseDependencies(tc.Required, tc.Optional).StripAnsi()
			require.Equal(t, tc.Expected, actual)
		})
	}
}
