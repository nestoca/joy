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

func TestResolveProjectTemplates(t *testing.T) {
	cases := []struct {
		name                    string
		project                 *v1alpha1.Project
		catalogProjectTemplates map[string]string
		expected                map[string]string
	}{
		{
			name:    "Just catalog-level project templates",
			project: &v1alpha1.Project{},
			catalogProjectTemplates: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Just project-level project templates",
			project: &v1alpha1.Project{
				Spec: v1alpha1.ProjectSpec{
					Links: map[string]string{
						"link1": "template1",
						"link2": "template2",
					},
				},
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Project links inherit catalog project links",
			project: &v1alpha1.Project{
				Spec: v1alpha1.ProjectSpec{
					Links: map[string]string{
						"project-link1":         "project-template1",
						"catalog-project-link2": "catalog-project-template2-override",
					},
				},
			},
			catalogProjectTemplates: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2",
			},
			expected: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2-override",
				"project-link1":         "project-template1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := resolveProjectTemplates(tc.project, tc.catalogProjectTemplates)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestResolveReleaseTemplates(t *testing.T) {
	cases := []struct {
		name                    string
		release                 *v1alpha1.Release
		catalogProjectTemplates map[string]string
		catalogReleaseTemplates map[string]string
		expected                map[string]string
	}{
		{
			name: "Just catalog-level project templates",
			release: &v1alpha1.Release{
				Spec:    v1alpha1.ReleaseSpec{},
				Project: &v1alpha1.Project{},
			},
			catalogProjectTemplates: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Just catalog-level release templates",
			release: &v1alpha1.Release{
				Spec:    v1alpha1.ReleaseSpec{},
				Project: &v1alpha1.Project{},
			},
			catalogReleaseTemplates: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Just project-level project templates",
			release: &v1alpha1.Release{
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Links: map[string]string{
							"link1": "template1",
							"link2": "template2",
						},
					},
				},
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Just project-level release templates",
			release: &v1alpha1.Release{
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						ReleaseLinks: map[string]string{
							"link1": "template1",
							"link2": "template2",
						},
					},
				},
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Just release templates",
			release: &v1alpha1.Release{
				Spec: v1alpha1.ReleaseSpec{
					Links: map[string]string{
						"link1": "template1",
						"link2": "template2",
					},
				},
				Project: &v1alpha1.Project{},
			},
			expected: map[string]string{
				"link1": "template1",
				"link2": "template2",
			},
		},
		{
			name: "Catalog release links inherit catalog project links",
			release: &v1alpha1.Release{
				Project: &v1alpha1.Project{},
			},
			catalogProjectTemplates: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2",
			},
			catalogReleaseTemplates: map[string]string{
				"catalog-project-link2": "catalog-project-template2-override",
				"catalog-release-link1": "catalog-release-template1",
			},
			expected: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2-override",
				"catalog-release-link1": "catalog-release-template1",
			},
		},
		{
			name: "Project links inherit catalog project links",
			release: &v1alpha1.Release{
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Links: map[string]string{
							"project-link1":         "project-template1",
							"catalog-project-link2": "catalog-project-template2-override",
						},
					},
				},
			},
			catalogProjectTemplates: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2",
			},
			expected: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2-override",
				"project-link1":         "project-template1",
			},
		},
		{
			name: "Release links inherit project and catalog project and release links",
			release: &v1alpha1.Release{
				Spec: v1alpha1.ReleaseSpec{
					Links: map[string]string{
						"release-link1": "release-template1",
					},
				},
				Project: &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Links: map[string]string{
							"project-link1": "project-template1",
						},
					},
				},
			},
			catalogProjectTemplates: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2",
			},
			catalogReleaseTemplates: map[string]string{
				"catalog-release-link1": "catalog-release-template1",
				"catalog-release-link2": "catalog-release-template2",
			},
			expected: map[string]string{
				"catalog-project-link1": "catalog-project-template1",
				"catalog-project-link2": "catalog-project-template2",
				"catalog-release-link1": "catalog-release-template1",
				"catalog-release-link2": "catalog-release-template2",
				"project-link1":         "project-template1",
				"release-link1":         "release-template1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := resolveReleaseTemplates(tc.release, tc.catalogProjectTemplates, tc.catalogReleaseTemplates)
			require.Equal(t, tc.expected, actual)
		})
	}
}
