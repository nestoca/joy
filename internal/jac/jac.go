package jac

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nestoca/survey/v2"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

var dependency = &dependencies.Dependency{
	Command:    "jac",
	Url:        "https://github.com/nestoca/jac",
	IsRequired: false,
	RequiredBy: []string{"project owners", "release owners"},
}

func init() {
	dependencies.Add(dependency)
}

func ListProjectPeople(catalogDir string, extraArgs []string) error {
	if err := dependency.MustBeInstalled(); err != nil {
		return err
	}

	cat, err := catalog.Load(catalog.LoadOpts{Dir: catalogDir})
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Select project
	selectedProject, err := selectProject(cat.Projects)
	if err != nil {
		return err
	}

	return listPeopleWithGroups(selectedProject.Spec.Owners, extraArgs)
}

func ListReleasePeople(catalogDir string, extraArgs []string) error {
	if err := dependency.MustBeInstalled(); err != nil {
		return err
	}

	cat, err := catalog.Load(catalog.LoadOpts{Dir: catalogDir})
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Select cross-release
	selectedCrossRelease, err := selectCrossRelease(cat.Releases.Items)
	if err != nil {
		return err
	}

	// Find project of first release within cross-release that has a project
	var proj *v1alpha1.Project
	for _, rel := range selectedCrossRelease.Releases {
		if rel != nil && rel.Project != nil {
			proj = rel.Project
			break
		}
	}
	if proj == nil {
		fmt.Printf("🤷 Release %s has no associated project, please set %s property.\n", style.Resource(selectedCrossRelease.Name), style.Code("spec.project"))
		return nil
	}

	// List owners
	if len(proj.Spec.Owners) == 0 {
		fmt.Printf("🤷 Project %s has no associated owners, please set %s property.\n", style.Resource(proj.Name), style.Code("spec.owners"))
		return nil
	}
	return listPeopleWithGroups(proj.Spec.Owners, extraArgs)
}

func listPeopleWithGroups(groups []string, extraArgs []string) error {
	args := []string{"people", "--group", strings.Join(groups, ",")}
	args = append(args, extraArgs...)
	cmd := exec.Command("jac", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running jac command: %w", err)
	}
	return nil
}

func selectProject(projects []*v1alpha1.Project) (*v1alpha1.Project, error) {
	var selectedIndex int
	err := survey.AskOne(&survey.Select{
		Message: "Select project:",
		Options: projectNames(projects),
	},
		&selectedIndex,
		survey.WithPageSize(10),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for project: %w", err)
	}
	return projects[selectedIndex], nil
}

func projectNames(projects []*v1alpha1.Project) []string {
	var projectNames []string
	for _, project := range projects {
		projectNames = append(projectNames, project.Name)
	}
	return projectNames
}

func selectCrossRelease(releases []*cross.Release) (*cross.Release, error) {
	var selectedIndex int
	err := survey.AskOne(&survey.Select{
		Message: "Select release:",
		Options: releaseNames(releases),
	},
		&selectedIndex,
		survey.WithPageSize(10),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for release: %w", err)
	}
	return releases[selectedIndex], nil
}

func releaseNames(releases []*cross.Release) []string {
	var releaseNames []string
	for _, release := range releases {
		releaseNames = append(releaseNames, release.Name)
	}
	return releaseNames
}
