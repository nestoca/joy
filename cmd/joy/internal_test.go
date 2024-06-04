package main

import (
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInternal(t *testing.T) {
	if ok, _ := strconv.ParseBool(os.Getenv("INTERNAL_TESTS")); !ok {
		t.Skip("Internal tests")
	}

	configDir, err := os.MkdirTemp("", "test-config-root-*")
	require.NoError(t, err)

	catalogDir := os.Getenv("CATALOG_DIR")
	require.NotEmpty(t, catalogDir, "catalog repo must be provided")

	binaryFile, err := os.CreateTemp("", "joy-test-*")
	require.NoError(t, err)
	require.NoError(t, binaryFile.Close())

	require.NoError(t, exec.Command("go", "build", "-o", binaryFile.Name(), ".").Run())

	command := func(args ...string) ([]byte, error) {
		args = append(args, "--skip-dev-check", "--config-dir="+configDir)
		return exec.Command(binaryFile.Name(), args...).CombinedOutput()
	}

	t.Run("setup", func(t *testing.T) {
		out, err := command("setup", "--catalog-dir="+catalogDir)
		require.NoError(t, err, string(out))
	})

	t.Run("joy rel validate", func(t *testing.T) {
		_, err := command("release", "validate")
		require.NoError(t, err)
	})

	t.Run("joy rel ls", func(t *testing.T) {
		_, err := command("rel", "ls")
		require.NoError(t, err)
	})

	t.Run("joy rel sel all", func(t *testing.T) {
		_, err := command("rel", "sel", "--all")
		require.NoError(t, err)
	})

	t.Run("joy env select all", func(t *testing.T) {
		_, err := command("env", "sel", "--all")
		require.NoError(t, err)
	})

	t.Run("joy rel prom dry run", func(t *testing.T) {
		_, err := command(
			"rel", "prom", "canary",
			"--no-prompt", "--draft", "--dry-run",
			"--source=qa", "--target=production",
		)
		require.NoError(t, err)
	})
}
