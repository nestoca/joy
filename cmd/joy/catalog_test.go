package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal"
)

func TestCatalogDir_Output(t *testing.T) {
	tmpDir := t.TempDir()
	var out bytes.Buffer
	err := run(RunParams{
		version:       "v1.0.0",
		args:          []string{"--skip-dev-check", "--skip-version-check", "--catalog-dir", tmpDir, "--config-dir", tmpDir, "catalog", "dir"},
		io:            internal.IO{Out: &out, Err: os.Stderr, In: os.Stdin},
		preRunConfigs: make(PreRunConfigs),
	})
	require.NoError(t, err)

	want, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	want = filepath.Clean(want)
	require.Equal(t, want, out.String())
	require.NotContains(t, out.String(), "\n", "output must not end with a newline")
}
