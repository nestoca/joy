package diagnostics

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
)

func TestConfigDiagnostics(t *testing.T) {
	cases := []struct {
		Name     string
		Config   *config.Config
		Stat     func(string) (fs.FileInfo, error)
		Expected Group
	}{
		{
			Name:   "happy",
			Config: &config.Config{FilePath: ".joyrc"},
			Stat:   func(string) (fs.FileInfo, error) { return nil, nil },
			Expected: Group{
				Title: "Config",
				Messages: Messages{
					{Type: "success", Value: "File exists: .joyrc"},
					{Type: "info", Value: "Selected environments: <all>"},
					{Type: "info", Value: "Selected releases: <all>"},
				},
				topLevel: true,
			},
		},
		{
			Name:   "file not exists",
			Config: &config.Config{FilePath: ".joyrc"},
			Stat:   func(string) (fs.FileInfo, error) { return nil, os.ErrNotExist },
			Expected: Group{
				Title: "Config",
				Messages: Messages{
					{Type: "failed", Value: "File does not exist: .joyrc"},
				},
				topLevel: true,
			},
		},
		{
			Name:   "fail to stat file",
			Config: &config.Config{FilePath: ".joyrc"},
			Stat:   func(string) (fs.FileInfo, error) { return nil, errors.New("corrupted disk!") },
			Expected: Group{
				Title: "Config",
				Messages: Messages{
					{Type: "failed", Value: "Failed to get config file: corrupted disk!"},
				},
				topLevel: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(
				t,
				tc.Expected,
				diagnoseConfig(tc.Config, ConfigOpts{Stat: tc.Stat}).StripAnsi(),
			)
		})
	}
}
