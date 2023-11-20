package diagnostics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
)

func TestExecutableDiagnostics(t *testing.T) {
	cases := []struct {
		Name             string
		Version          string
		MinVersion       string
		LookupExecutable func() (string, error)
		AbsolutePath     func(string) (string, error)
		Expected         Group
	}{
		{
			Name:             "happy",
			Version:          "v1.0.0",
			MinVersion:       "v1.0.0",
			LookupExecutable: func() (string, error) { return "binary_path", nil },
			AbsolutePath:     func(string) (string, error) { return "absolute_binary_path", nil },
			Expected: Group{
				Title:    "Executable",
				toplevel: true,
				Messages: Messages{
					{Type: "info", Value: "Version: v1.0.0"},
					{Type: "success", Value: "Version meets minimum of v1.0.0 required by catalog"},
					{Type: "info", Value: "File path: absolute_binary_path"},
				},
			},
		},
		{
			Name:       "invalid version",
			Version:    "dev-build",
			MinVersion: "v1.0.0",
			Expected: Group{
				Title:    "Executable",
				toplevel: true,
				Messages: Messages{
					{
						Type:  "info",
						Value: "Version: dev-build",
					},
					{
						Type:  "warning",
						Value: "Version is not in semver format and cannot be compared with minimum of v1.0.0 required by catalog",
					},
				},
			},
		},
		{
			Name:       "does not meet minimum version",
			Version:    "v1.0.0",
			MinVersion: "v2.0.0",
			Expected: Group{
				Title:    "Executable",
				toplevel: true,
				Messages: Messages{
					{
						Type:  "info",
						Value: "Version: v1.0.0",
					},
					{
						Type:  "failed",
						Value: "Version does not meet minimum of v2.0.0 required by catalog",
						Details: Messages{
							{Type: "hint", Value: "Update joy using: brew upgrade joy"},
						},
					},
				},
			},
		},
		{
			Name:             "failed getting executable path",
			Version:          "v1.0.0",
			MinVersion:       "v1.0.0",
			LookupExecutable: func() (string, error) { return "", errors.New("exe not in path") },
			Expected: Group{
				Title: "Executable", Messages: Messages{
					{Type: "info", Value: "Version: v1.0.0"},
					{Type: "success", Value: "Version meets minimum of v1.0.0 required by catalog"},
					{Type: "failed", Value: "failed to get executable path: exe not in path"},
				},
				toplevel: true,
			},
		},
		{
			Name:             "failed getting absolute path",
			Version:          "v1.0.0",
			MinVersion:       "v1.0.0",
			LookupExecutable: func() (string, error) { return "binary_path", nil },
			AbsolutePath:     func(s string) (string, error) { return "", errors.New("FS make no sense!") },
			Expected: Group{
				Title: "Executable", Messages: Messages{
					{Type: "info", Value: "Version: v1.0.0"},
					{Type: "success", Value: "Version meets minimum of v1.0.0 required by catalog"},
					{Type: "failed", Value: "failed to get absolute path of executable: FS make no sense!"},
				},
				toplevel: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			opts := ExecutableOptions{
				LookupExectuble: tc.LookupExecutable,
				AbsolutePath:    tc.AbsolutePath,
			}
			require.Equal(
				t,
				tc.Expected,
				diagnoseExecutable(&config.Config{MinVersion: tc.MinVersion}, tc.Version, opts).StripAnsi(),
			)
		})
	}
}
