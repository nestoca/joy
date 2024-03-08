package links

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nestoca/survey/v2"
	"github.com/nestoca/survey/v2/core"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

func GetEnvironmentLinks(provider Provider, cat *catalog.Catalog, envName string) (map[string]string, error) {
	env, err := getOrSelectEnvironment(cat, envName)
	if err != nil {
		return nil, err
	}

	links, err := provider.GetEnvironmentLinks(env)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func GetProjectLinks(provider Provider, cat *catalog.Catalog, projectName string) (map[string]string, error) {
	project, err := getOrSelectProject(cat, projectName)
	if err != nil {
		return nil, err
	}

	links, err := provider.GetProjectLinks(project)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func GetReleaseLinks(provider Provider, cat *catalog.Catalog, envName, releaseName string) (map[string]string, error) {
	env, err := getOrSelectEnvironment(cat, envName)
	if err != nil {
		return nil, err
	}

	releaseName, err = getOrSelectReleaseName(cat, releaseName)
	if err != nil {
		return nil, err
	}

	release, err := cat.Releases.GetEnvironmentRelease(env, releaseName)
	if err != nil {
		return nil, err
	}

	links, err := provider.GetReleaseLinks(release)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func PrintLinks(links map[string]string, linkName string) error {
	if linkName != "" {
		linkUrl := links[linkName]
		if linkUrl == "" {
			return getLinkNotFoundError(linkName, links)
		}
		fmt.Print(linkUrl)
		return nil
	}

	return printLinksTable(links)
}

func printLinksTable(links map[string]string) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"NAME", "URL"})

	linkNames := getSortedLinkNames(links)
	for _, linkName := range linkNames {
		linkUrl := links[linkName]
		t.AppendRow(table.Row{style.Resource(linkName), linkUrl})
	}

	_, err := fmt.Println(t.Render())
	return err
}

func GetOrSelectLinkUrl(links map[string]string, linkName string) (string, error) {
	var linkUrl string
	if linkName == "" {
		linkUrl, err := selectLinkUrl(links)
		if err != nil {
			return "", err
		}
		return linkUrl, nil
	}

	linkUrl = links[linkName]
	if linkUrl == "" {
		return "", getLinkNotFoundError(linkName, links)
	}
	return linkUrl, nil
}

func getOrSelectEnvironment(cat *catalog.Catalog, envName string) (*v1alpha1.Environment, error) {
	if envName == "" {
		return environment.SelectSingle(cat.Environments, nil, "Select environment")
	}

	return v1alpha1.GetEnvironmentByName(cat.Environments, envName)
}

func getOrSelectProject(cat *catalog.Catalog, projectName string) (*v1alpha1.Project, error) {
	if projectName == "" {
		return selectProject(cat)
	}

	for _, project := range cat.Projects {
		if project.Name == projectName {
			return project, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", projectName)
}

func selectProject(cat *catalog.Catalog) (*v1alpha1.Project, error) {
	var choices []string
	for _, project := range cat.Projects {
		choices = append(choices, project.Name)
	}
	prompt := &survey.Select{
		Message: "Select project",
		Options: choices,
	}
	questions := []*survey.Question{{
		Prompt: prompt,
	}}
	var index int
	err := survey.Ask(questions,
		&index,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required))
	if err != nil {
		return nil, fmt.Errorf("selecting release: %w", err)
	}
	return cat.Projects[index], nil
}

func getOrSelectReleaseName(cat *catalog.Catalog, releaseName string) (string, error) {
	if releaseName != "" {
		return releaseName, nil
	}

	var choices []string
	for _, release := range cat.Releases.Items {
		choices = append(choices, release.Name)
	}
	prompt := &survey.Select{
		Message: "Select release",
		Options: choices,
	}
	questions := []*survey.Question{{
		Prompt: prompt,
	}}
	err := survey.Ask(questions,
		&releaseName,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required))
	if err != nil {
		return "", fmt.Errorf("selecting release: %w", err)
	}
	return releaseName, nil
}

func getSortedLinkNames(links map[string]string) []string {
	linkNames := make([]string, 0, len(links))
	for linkName := range links {
		linkNames = append(linkNames, linkName)
	}
	sort.Strings(linkNames)
	return linkNames
}

func getLinkNotFoundError(linkName string, links map[string]string) error {
	linkNames := getSortedLinkNames(links)
	return fmt.Errorf("link %q not found in links: %s", linkName, strings.Join(linkNames, ", "))
}

func selectLinkUrl(links map[string]string) (string, error) {
	linkNames := getSortedLinkNames(links)
	var choices []string
	for _, linkName := range linkNames {
		linkUrl := links[linkName]
		choices = append(choices, fmt.Sprintf("%s: %s", style.Resource(linkName), style.SecondaryInfo(linkUrl)))
	}

	// Transform allows to show only link name after user confirms selection,
	// instead of the full string with link name and URL.
	transform := func(ans interface{}) interface{} {
		answer := ans.(core.OptionAnswer)
		answer.Value = linkNames[answer.Index]
		return answer
	}

	prompt := &survey.Select{
		Message: "Select link",
		Options: choices,
	}
	questions := []*survey.Question{{
		Prompt:    prompt,
		Transform: transform,
	}}
	var linkIndex int
	err := survey.Ask(questions,
		&linkIndex,
		survey.WithKeepFilter(true),
		survey.WithValidator(survey.Required))
	if err != nil {
		return "", fmt.Errorf("selecting link: %w", err)
	}
	return links[linkNames[linkIndex]], nil
}

func OpenUrl(url string) error {
	fmt.Printf("🔗 Opening: %s\n", style.SecondaryInfo(url))

	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}

	args = append(args, url)
	err := exec.Command(cmd, args...).Run()
	if err != nil {
		return fmt.Errorf("opening URL %q: %w", url, err)
	}
	return nil
}
