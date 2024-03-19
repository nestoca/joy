package links

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/info"
)

func TestGetEnvironmentLinks(t *testing.T) {
	cases := []struct {
		name      string
		templates map[string]string
		expected  map[string]string
	}{
		{
			name: "Environment name",
			templates: map[string]string{
				"test": "test/{{ .Environment.Name }}",
			},
			expected: map[string]string{
				"test": "test/staging",
			},
		},
	}

	mockedProvider := &info.ProviderMock{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewProvider(mockedProvider, config.Templates{
				Environment: config.EnvironmentTemplates{Links: tc.templates},
			})
			actual, err := provider.GetEnvironmentLinks(&v1alpha1.Environment{
				EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
					Name: "staging",
				},
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetProjectLinks(t *testing.T) {
	cases := []struct {
		name      string
		templates map[string]string
		expected  map[string]string
	}{
		{
			name: "Project name",
			templates: map[string]string{
				"test": "test/{{ .Project.Name }}",
			},
			expected: map[string]string{
				"test": "test/my-project",
			},
		},
		{
			name: "Project repo",
			templates: map[string]string{
				"test": "test/{{ .Repository }}",
			},
			expected: map[string]string{
				"test": "test/my-project-repo",
			},
		},
	}

	mockedProvider := &info.ProviderMock{
		GetProjectRepositoryFunc: func(project *v1alpha1.Project) string {
			return "my-project-repo"
		},
	}

	project := &v1alpha1.Project{
		ProjectMetadata: v1alpha1.ProjectMetadata{
			Name: "my-project",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewProvider(mockedProvider, config.Templates{
				Project: config.ProjectTemplates{Links: tc.templates},
			})
			actual, err := provider.GetProjectLinks(project)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetReleaseLinks(t *testing.T) {
	cases := []struct {
		name      string
		templates map[string]string
		expected  map[string]string
	}{
		{
			name: "Release name",
			templates: map[string]string{
				"test": "test/{{ .Release.Name }}",
			},
			expected: map[string]string{
				"test": "test/my-release",
			},
		},
		{
			name: "Project name",
			templates: map[string]string{
				"test": "test/{{ .Project.Name }}",
			},
			expected: map[string]string{
				"test": "test/my-project",
			},
		},
		{
			name: "Environment name",
			templates: map[string]string{
				"test": "test/{{ .Environment.Name }}",
			},
			expected: map[string]string{
				"test": "test/staging",
			},
		},
		{
			name: "Project repo",
			templates: map[string]string{
				"test": "test/{{ .Repository }}",
			},
			expected: map[string]string{
				"test": "test/my-project-repo",
			},
		},
		{
			name: "Git tag",
			templates: map[string]string{
				"test": "test/{{ .GitTag }}",
			},
			expected: map[string]string{
				"test": "test/v1.2.3",
			},
		},
	}

	mockedProvider := &info.ProviderMock{
		GetProjectRepositoryFunc: func(project *v1alpha1.Project) string {
			return "my-project-repo"
		},
		GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
			return "v1.2.3", nil
		},
	}

	env := &v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
			Name: "staging",
		},
	}
	project := &v1alpha1.Project{
		ProjectMetadata: v1alpha1.ProjectMetadata{
			Name: "my-project",
		},
	}
	rel := &v1alpha1.Release{
		ReleaseMetadata: v1alpha1.ReleaseMetadata{
			Name: "my-release",
		},
		Environment: env,
		Project:     project,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewProvider(mockedProvider, config.Templates{
				Release: config.ReleaseTemplates{Links: tc.templates},
			})
			actual, err := provider.GetReleaseLinks(rel)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
