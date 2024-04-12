package catalog

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
)

type Builder struct {
	Catalog
	t *testing.T
}

func NewBuilder(t *testing.T) Builder {
	return Builder{
		t: t,
	}
}

func (b *Builder) AddEnvironment(name string, f func(e *v1alpha1.Environment)) *v1alpha1.Environment {
	environment := v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{Name: name},
	}
	if f != nil {
		f(&environment)
	}
	b.Environments = append(b.Environments, &environment)
	return &environment
}

func (b *Builder) AddProject(name string, f func(p *v1alpha1.Project)) *v1alpha1.Project {
	project := v1alpha1.Project{
		ProjectMetadata: v1alpha1.ProjectMetadata{Name: name},
	}
	if f != nil {
		f(&project)
	}
	b.Projects = append(b.Projects, &project)
	return &project
}

func (b *Builder) AddRelease(env *v1alpha1.Environment, project *v1alpha1.Project, version string, f func(r *v1alpha1.Release)) *v1alpha1.Release {
	b.t.Helper()
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
		for _, crossRel := range b.Releases.Items {
			if crossRel.Name == rel.Name {
				return crossRel
			}
		}

		crossRel := &cross.Release{
			Name:     rel.Name,
			Releases: make([]*v1alpha1.Release, len(b.Environments)),
		}
		b.Releases.Items = append(b.Releases.Items, crossRel)
		return crossRel
	}()

	envIndex := slices.Index(b.Environments, env)
	require.NotEqual(b.t, -1, envIndex, "environment "+env.Name+" not found")
	crossRel.Releases[envIndex] = &rel

	return &rel
}

func (b *Builder) Build() *Catalog {
	b.Releases.Environments = b.Environments
	return &b.Catalog
}
