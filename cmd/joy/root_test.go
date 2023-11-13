package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
)

func TestRootVersions(t *testing.T) {
	// For the version command to work that we use for testing versions,
	// we need to trick joy into thinking there is a catalog setup somewhere.
	catalogDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(catalogDir, "environments"), 0o755))

	cases := []struct {
		Name          string
		MinVersion    string
		Version       string
		ExpectedError string
	}{
		{
			Name:          "less than min",
			MinVersion:    "v1.0.0",
			Version:       "v0.0.7",
			ExpectedError: `current version "v0.0.7" is less than required minimum version "v1.0.0". Please update joy`,
		},
		{
			Name:          "prerelease",
			MinVersion:    "v1.0.0",
			Version:       "v1.0.0-alpha",
			ExpectedError: `current version "v1.0.0-alpha" is less than required minimum version "v1.0.0". Please update joy`,
		},
		{
			Name:       "equal to min",
			MinVersion: "v1.0.0",
			Version:    "v1.0.0",
		},
		{
			Name:       "greater than min",
			MinVersion: "v1",
			Version:    "v1.2.3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := config.ToContext(context.Background(), &config.Config{
				MinVersion: tc.MinVersion,
				CatalogDir: catalogDir,
			})

			cmd := NewRootCmd(tc.Version)
			cmd.SetArgs([]string{"version"})

			var buffer bytes.Buffer
			cmd.SetOut(&buffer)

			err := cmd.ExecuteContext(ctx)
			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.Version, strings.TrimSpace(buffer.String()))
			}
		})
	}
}
