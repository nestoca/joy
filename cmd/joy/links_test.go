package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/pkg/catalog"
)

func executeLinksCommand(t *testing.T, cmd *cobra.Command, args ...string) string {
	cfg := config.Config{
		GitHubOrganization: "acme",
		Templates: config.Templates{
			Environment: config.EnvironmentTemplates{
				Links: map[string]string{
					"cd": "https://argo-cd.acme.com/applications/{{ .Environment.Name }}",
				},
			},
			Project: config.ProjectTemplates{
				Links: map[string]string{
					"repo":    "https://github.com/{{ .Repository }}",
					"actions": "https://github.com/{{ .Repository }}/actions",
					"pulls":   "https://github.com/{{ .Repository }}/pulls",
				},
			},
			Release: config.ReleaseTemplates{
				Links: map[string]string{
					"tag": "https://github.com/{{ .Repository }}/releases/tag/{{ .GitTag }}",
				},
			},
		},
	}

	cat := newTestCatalog(t, newTestCatalogParams{
		getProject: func(project *v1alpha1.Project) *v1alpha1.Project {
			project.Spec.Links = map[string]string{
				"repo":         "https://github.com/{{ .Repository }}-project-override",
				"project-link": "acme.com/projects/{{ .Project.Name }}",
			}
			project.Spec.ReleaseLinks = map[string]string{
				"project-release-link":             "acme.com/projects/{{ .Project.Name }}/releases/{{ .Release.Name }}",
				"project-release-link-to-override": "acme.com/projects/{{ .Project.Name }}/releases/{{ .Release.Name }}",
			}
			return project
		},
		getRelease: func(release *v1alpha1.Release) *v1alpha1.Release {
			release.Spec.Links = map[string]string{
				"release-link":                     "acme.com/releases/{{ .Release.Name }}",
				"project-release-link-to-override": "acme.com/projects/{{ .Project.Name }}/releases/{{ .Release.Name }}-release-override",
			}
			return release
		},
	})

	ctx := config.ToContext(catalog.ToContext(context.Background(), cat), &cfg)

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
│ tag                              │ https://github.com/acme/my-project/releases/tag/1.2.3             │
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
