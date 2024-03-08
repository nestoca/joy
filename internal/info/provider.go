//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package info

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nestoca/joy/internal/git"

	"github.com/nestoca/joy/internal/github"

	"github.com/nestoca/joy/api/v1alpha1"
)

type Provider interface {
	GetProjectRepository(project *v1alpha1.Project) string
	GetProjectSourceDir(project *v1alpha1.Project) (string, error)
	GetCommitsMetadata(projectDir, fromTag, toTag string) ([]*CommitMetadata, error)
	GetCommitsGitHubAuthors(project *v1alpha1.Project, fromTag, toTag string) (map[string]string, error)
	GetCodeOwners(projectDir string) ([]string, error)
	GetReleaseGitTag(release *v1alpha1.Release) (string, error)
}

type CommitMetadata struct {
	Sha     string
	Author  string
	Message string
}

type defaultProvider struct {
	gitHubOrganization    string
	defaultGitTagTemplate string
	repositoriesCacheDir  string
	joyCacheDir           string
}

func NewProvider(gitHubOrganization, defaultGitTagTemplate, repositoriesCacheDir, joyCacheDir string) Provider {
	return &defaultProvider{
		gitHubOrganization:    gitHubOrganization,
		defaultGitTagTemplate: defaultGitTagTemplate,
		repositoriesCacheDir:  repositoriesCacheDir,
		joyCacheDir:           joyCacheDir,
	}
}

func (p *defaultProvider) GetProjectRepository(proj *v1alpha1.Project) string {
	if proj.Spec.Repository != "" {
		return proj.Spec.Repository
	}
	return fmt.Sprintf("%s/%s", p.gitHubOrganization, proj.Name)
}

func (p *defaultProvider) GetProjectSourceDir(proj *v1alpha1.Project) (string, error) {
	cacheDir := cmp.Or(p.repositoriesCacheDir, filepath.Join(p.joyCacheDir, "src"))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("creating project cache dir %q: %w", cacheDir, err)
	}

	repository := proj.Spec.Repository
	if repository == "" {
		repository = fmt.Sprintf("%s/%s", p.gitHubOrganization, proj.Name)
	}

	repoDir := filepath.Join(cacheDir, path.Base(repository))
	if _, err := os.Stat(repoDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		cloneOptions := github.CloneOptions{
			RepoURL: repository,
			OutDir:  repoDir,
		}
		if err := github.Clone(cacheDir, cloneOptions); err != nil {
			return "", fmt.Errorf("cloning project %s from repo %q: %w", proj.Name, repository, err)
		}
		return repoDir, nil
	}

	if err := git.Fetch(repoDir); err != nil {
		return "", fmt.Errorf("fetching git commits for cached project %s cloned at %q: %w", proj.Name, repoDir, err)
	}

	if err := git.FetchTags(repoDir); err != nil {
		return "", fmt.Errorf("fetching git tags for cached project %s cloned at %q: %w", proj.Name, repoDir, err)
	}
	return repoDir, nil
}

func (p *defaultProvider) GetReleaseGitTag(release *v1alpha1.Release) (string, error) {
	gitTagTemplate := cmp.Or(release.Project.Spec.GitTagTemplate, p.defaultGitTagTemplate)
	if gitTagTemplate == "" {
		return release.Spec.Version, nil
	}

	tmpl, err := template.New("").Parse(gitTagTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing git tag template %q: %w", gitTagTemplate, err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, struct{ Release *v1alpha1.Release }{release}); err != nil {
		return "", fmt.Errorf("executing git tag template %q: %w", gitTagTemplate, err)
	}

	return buffer.String(), nil
}

type commitInfo struct {
	Sha   string `json:"sha"`
	Login string `json:"login"`
}

func (p *defaultProvider) GetCommitsMetadata(dir, from, to string) ([]*CommitMetadata, error) {
	gitArgs := []string{"log", "--pretty=format:%H%n%an%n%s%n---END---%n", from + ".." + to}
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing %q: %s", "git "+strings.Join(gitArgs, " "), output)
	}

	var commits []*CommitMetadata
	commitItems := strings.Split(string(output), "\n---END---\n")
	for _, commitItem := range commitItems {
		commitItem = strings.TrimSpace(commitItem)
		if commitItem == "" {
			continue
		}
		lines := strings.SplitN(commitItem, "\n", 3)
		if len(lines) < 3 {
			return nil, fmt.Errorf("malformed commit output: %q", commitItem)
		}
		sha := lines[0]
		author := lines[1]
		message := lines[2]

		commits = append(commits, &CommitMetadata{
			Sha:     sha,
			Author:  author,
			Message: message,
		})
	}
	return commits, nil
}

func (p *defaultProvider) GetCommitsGitHubAuthors(project *v1alpha1.Project, fromTag, toTag string) (map[string]string, error) {
	repository := project.Spec.Repository
	if repository == "" {
		repository = fmt.Sprintf("%s/%s", p.gitHubOrganization, project.Name)
	}

	ghPath := fmt.Sprintf("repos/%s/compare/%s...%s", repository, fromTag, toTag)
	data, err := github.ExecuteAndGetOutput(".", "api", "--jq", "[.commits[] | {sha: .sha, login: (.author.login // .committer.login)}]", ghPath)
	if err != nil {
		return nil, fmt.Errorf("getting commits GitHub authors: %w", err)
	}

	var commits []commitInfo
	err = json.Unmarshal([]byte(data), &commits)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling commits GitHub authors: %w", err)
	}

	authors := make(map[string]string)
	for _, commit := range commits {
		authors[commit.Sha] = commit.Login
	}
	return authors, nil
}

func (p *defaultProvider) GetCodeOwners(dir string) ([]string, error) {
	var owners []string
	for _, relativeFilepath := range []string{".github/CODEOWNERS", "CODEOWNERS"} {
		fullPath := dir + "/" + relativeFilepath
		_, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		owners = append(owners, parseCodeOwners(string(content))...)
	}
	return owners, nil
}

func parseCodeOwners(content string) []string {
	var owners []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		for _, owner := range parts[1:] {
			if owner == "" {
				continue
			}
			owners = append(owners, strings.TrimLeft(owner, "@"))
		}
	}

	return owners
}

func (p *defaultProvider) GetGitTag(release *v1alpha1.Release, defaultGitTagTemplate string) (string, error) {
	gitTagTemplate := cmp.Or(release.Project.Spec.GitTagTemplate, defaultGitTagTemplate)
	if gitTagTemplate == "" {
		return release.Spec.Version, nil
	}

	tmpl, err := template.New("").Parse(gitTagTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing git tag template %q: %w", gitTagTemplate, err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, struct{ Release *v1alpha1.Release }{release}); err != nil {
		return "", fmt.Errorf("executing git tag template %q: %w", gitTagTemplate, err)
	}

	return buffer.String(), nil
}
