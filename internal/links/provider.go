package links

import (
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/info"
)

//go:generate moq -stub -out ./provider_mock.go . Provider
type Provider interface {
	GetEnvironmentLinks(environment *v1alpha1.Environment) (map[string]string, error)
	GetProjectLinks(project *v1alpha1.Project) (map[string]string, error)
	GetReleaseLinks(release *v1alpha1.Release) (map[string]string, error)
}

func NewProvider(infoProvider info.Provider, templates config.Templates) Provider {
	return &provider{
		infoProvider: infoProvider,
		templates:    templates,
	}
}

type provider struct {
	infoProvider info.Provider
	templates    config.Templates
}

func (r *provider) GetEnvironmentLinks(environment *v1alpha1.Environment) (map[string]string, error) {
	templates := r.templates.Environment.Links
	links := make(map[string]string, len(templates))
	for name, tmpl := range templates {
		link, err := r.renderEnvironmentLink(tmpl, environment)
		if err != nil {
			return nil, fmt.Errorf("rendering environment link %s %q: %w", name, tmpl, err)
		}
		links[name] = link
	}
	return links, nil
}

func (r *provider) GetProjectLinks(project *v1alpha1.Project) (map[string]string, error) {
	templates := resolveProjectTemplates(project, r.templates.Project.Links)
	links := make(map[string]string, len(templates))
	for name, tmpl := range templates {
		link, err := r.renderProjectLink(tmpl, project)
		if err != nil {
			return nil, fmt.Errorf("rendering project link %s %q: %w", name, tmpl, err)
		}
		links[name] = link
	}
	return links, nil
}

func (r *provider) GetReleaseLinks(release *v1alpha1.Release) (map[string]string, error) {
	if release == nil {
		return nil, nil
	}

	templates := resolveReleaseTemplates(release, r.templates.Project.Links, r.templates.Release.Links)
	links := make(map[string]string, len(templates))
	for name, tmpl := range templates {
		link, err := r.renderReleaseLink(tmpl, release)
		if err != nil {
			return nil, fmt.Errorf("rendering release link %s %q: %w", name, tmpl, err)
		}
		links[name] = link
	}
	return links, nil
}

func (r *provider) renderEnvironmentLink(linkTemplate string, environment *v1alpha1.Environment) (string, error) {
	return renderLink(linkTemplate, struct {
		Environment *v1alpha1.Environment
	}{
		Environment: environment,
	})
}

func (r *provider) renderProjectLink(linkTemplate string, project *v1alpha1.Project) (string, error) {
	return renderLink(linkTemplate, struct {
		Project    *v1alpha1.Project
		Repository string
	}{
		Project:    project,
		Repository: r.infoProvider.GetProjectRepository(project),
	})
}

func (r *provider) renderReleaseLink(linkTemplate string, release *v1alpha1.Release) (string, error) {
	if release == nil {
		return "", nil
	}

	gitTag, err := r.infoProvider.GetReleaseGitTag(release)
	if err != nil {
		return "", fmt.Errorf("getting release git tag: %w", err)
	}

	return renderLink(linkTemplate, struct {
		Environment *v1alpha1.Environment
		Project     *v1alpha1.Project
		Release     *v1alpha1.Release
		Repository  string
		GitTag      string
	}{
		Environment: release.Environment,
		Project:     release.Project,
		Release:     release,
		Repository:  r.infoProvider.GetProjectRepository(release.Project),
		GitTag:      gitTag,
	})
}

func renderLink(linkTemplate string, data any) (string, error) {
	tmpl, err := template.New("message").Funcs(sprig.FuncMap()).Parse(linkTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing link template %q: %w", linkTemplate, err)
	}

	var message strings.Builder
	if err := tmpl.Execute(&message, data); err != nil {
		return "", fmt.Errorf("executing link template %q: %w", linkTemplate, err)
	}
	return message.String(), nil
}

func resolveProjectTemplates(project *v1alpha1.Project, catalogProjectLinks map[string]string) map[string]string {
	links := make(map[string]string)
	maps.Copy(links, catalogProjectLinks)
	maps.Copy(links, project.Spec.Links)
	return links
}

func resolveReleaseTemplates(release *v1alpha1.Release, catalogProjectLinks map[string]string, catalogReleaseLinks map[string]string) map[string]string {
	links := make(map[string]string)
	maps.Copy(links, catalogProjectLinks)
	maps.Copy(links, catalogReleaseLinks)
	if release != nil {
		maps.Copy(links, release.Project.Spec.Links)
		maps.Copy(links, release.Project.Spec.ReleaseLinks)
		maps.Copy(links, release.Spec.Links)
	}
	return links
}
