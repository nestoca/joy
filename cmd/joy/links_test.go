package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/acarl005/stripansi"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/pkg/catalog"
)

func newContextWithCatalogAndConfig(t *testing.T) (context.Context, error) {
	catalogDir, err := filepath.Abs("test_data/links_catalog")
	require.NoError(t, err)

	cfg, err := config.Load(catalogDir, catalogDir)
	require.NoError(t, err)

	cat, err := catalog.Load(catalogDir, nil)
	require.NoError(t, err)

	return config.ToContext(catalog.ToContext(context.Background(), cat), cfg), nil
}

func executeLinksCommand(t *testing.T, cmd *cobra.Command, args ...string) string {
	ctx, err := newContextWithCatalogAndConfig(t)
	require.NoError(t, err)

	var buffer bytes.Buffer
	cmd.SetOut(&buffer)
	cmd.SetArgs(args)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Execute())
	actual := stripansi.Strip(buffer.String())
	return actual
}

func TestReleaseLinks(t *testing.T) {
	actual := executeLinksCommand(t, NewReleaseLinksCmd(), "--env", "staging", "my-release")
	expected := `╭──────────────────────────────────┬───────────────────────────────────────────────────────────────────╮
│ NAME                             │ URL                                                               │
├──────────────────────────────────┼───────────────────────────────────────────────────────────────────┤
│ actions                          │ https://github.com/acme/my-project/actions                        │
│ project-link                     │ acme.com/projects/my-project                                      │
│ project-release-link             │ acme.com/projects/my-project/releases/my-release                  │
│ project-release-link-to-override │ acme.com/projects/my-project/releases/my-release-release-override │
│ pulls                            │ https://github.com/acme/my-project/pulls                          │
│ release-link                     │ acme.com/releases/my-release                                      │
│ repo                             │ https://github.com/acme/my-project-project-override               │
│ tag                              │ https://github.com/acme/my-project/releases/tag/                  │
╰──────────────────────────────────┴───────────────────────────────────────────────────────────────────╯
`
	require.Equal(t, expected, actual)
}

func TestReleaseSpecificLink(t *testing.T) {
	actual := executeLinksCommand(t, NewReleaseLinksCmd(), "--env", "staging", "my-release", "actions")
	expected := "https://github.com/acme/my-project/actions"
	require.Equal(t, expected, actual)
}

func TestProjectLinks(t *testing.T) {
	actual := executeLinksCommand(t, NewProjectLinksCmd(), "my-project")
	expected := `╭──────────────┬─────────────────────────────────────────────────────╮
│ NAME         │ URL                                                 │
├──────────────┼─────────────────────────────────────────────────────┤
│ actions      │ https://github.com/acme/my-project/actions          │
│ project-link │ acme.com/projects/my-project                        │
│ pulls        │ https://github.com/acme/my-project/pulls            │
│ repo         │ https://github.com/acme/my-project-project-override │
╰──────────────┴─────────────────────────────────────────────────────╯
`
	require.Equal(t, expected, actual)
}

func TestProjectSpecificLink(t *testing.T) {
	actual := executeLinksCommand(t, NewProjectLinksCmd(), "my-project", "actions")
	expected := "https://github.com/acme/my-project/actions"
	require.Equal(t, expected, actual)
}

func TestEnvironmentLinks(t *testing.T) {
	actual := executeLinksCommand(t, NewEnvironmentLinksCmd(), "staging")
	expected := `╭──────┬───────────────────────────────────────────────╮
│ NAME │ URL                                           │
├──────┼───────────────────────────────────────────────┤
│ cd   │ https://argo-cd.acme.com/applications/staging │
╰──────┴───────────────────────────────────────────────╯
`
	require.Equal(t, expected, actual)
}

func TestEnvironmentSpecificLink(t *testing.T) {
	actual := executeLinksCommand(t, NewEnvironmentLinksCmd(), "staging", "cd")
	expected := "https://argo-cd.acme.com/applications/staging"
	require.Equal(t, expected, actual)
}
