package jac

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/catalog"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
	"strings"
)

func ListProjectPeople(extraArgs []string) error {
	err := ensureJacCliInstalled()
	if err != nil {
		return err
	}

	err = git.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return err
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		LoadProjects: true,
	}
	cat, err := catalog.Load(loadOpts)
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

func ListReleasePeople(extraArgs []string) error {
	err := ensureJacCliInstalled()
	if err != nil {
		return err
	}

	err = git.EnsureCleanAndUpToDateWorkingCopy()
	if err != nil {
		return err
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		LoadEnvs:     true,
		LoadReleases: true,
		LoadProjects: true,
		ResolveRefs:  true,
	}
	cat, err := catalog.Load(loadOpts)
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

func ensureJacCliInstalled() error {
	cmd := exec.Command("which", "jac")
	err := cmd.Run()
	if err != nil {
		fmt.Println("🤓 This command requires the jac cli.\nSee: https://github.com/nestoca/jac")
		return errors.New("missing jac cli dependency")
	}
	return nil
}

func selectProject(projects []*v1alpha1.Project) (*v1alpha1.Project, error) {
	var selectedIndex int
	err := survey.AskOne(&survey.Select{
		Message: "ConfigureSelection project:",
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
		Message: "ConfigureSelection release:",
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
