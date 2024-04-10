package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

type CatalogBuilder struct {
	t            *testing.T
	environments []*v1alpha1.Environment
	projects     []*v1alpha1.Project
	releases     []*cross.Release
}

func NewCatalogBuilder(t *testing.T) CatalogBuilder {
	return CatalogBuilder{
		t: t,
	}
}

func (b *CatalogBuilder) AddEnvironment(name string, f func(e *v1alpha1.Environment)) *v1alpha1.Environment {
	environment := v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: name},
	}
	if f != nil {
		f(&environment)
	}
	b.environments = append(b.environments, &environment)
	return &environment
}

func (b *CatalogBuilder) AddProject(name string, f func(p *v1alpha1.Project)) *v1alpha1.Project {
	project := v1alpha1.Project{
		ProjectMetadata: v1alpha1.ProjectMetadata{Name: name},
	}
	if f != nil {
		f(&project)
	}
	b.projects = append(b.projects, &project)
	return &project
}

func (b *CatalogBuilder) AddRelease(env *v1alpha1.Environment, project *v1alpha1.Project, version string, f func(r *v1alpha1.Release)) *v1alpha1.Release {
	rel := v1alpha1.Release{
		ReleaseMetadata: v1alpha1.ReleaseMetadata{
			Name: project.Name,
		},
		Spec: v1alpha1.ReleaseSpec{
			Project: project.Name,
			Version: version,
		},
		Project:     project,
		Environment: env,
	}
	if f != nil {
		f(&rel)
	}

	file, err := yml.NewFileFromObject("environments/"+env.Name+"/releases/"+project.Name+".yaml", 2, &rel)
	require.NoError(b.t, err)
	rel.File = file

	crossRel := func() *cross.Release {
		for _, r := range b.releases {
			if r.Name == rel.Name {
				return r
			}
		}

		crossRel := &cross.Release{
			Name:     rel.Name,
			Releases: make([]*v1alpha1.Release, len(b.environments)),
		}
		b.releases = append(b.releases, crossRel)
		return crossRel
	}()

	envIndex := func() int {
		for i, e := range b.environments {
			if e == env {
				return i
			}
		}
		return -1
	}()
	require.NotEqual(b.t, -1, envIndex, "environment "+env.Name+" not found")
	crossRel.Releases[envIndex] = &rel

	return &rel
}

func (b *CatalogBuilder) Build() *catalog.Catalog {
	return &catalog.Catalog{
		Environments: b.environments,
		Projects:     b.projects,
		Releases: cross.ReleaseList{
			Environments: b.environments,
			Items:        b.releases,
		},
	}
}
