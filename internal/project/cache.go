package project

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/nestoca/joy/internal/github"

	"github.com/nestoca/joy/api/v1alpha1"

	"github.com/nestoca/joy/internal/git"
)

func GetCachedSourceDir(proj *v1alpha1.Project, defaultGitHubOrganization, repositoriesDir, joyCache string) (string, error) {
	cacheDir := cmp.Or(repositoriesDir, filepath.Join(joyCache, "src"))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("creating project cache dir %q: %w", cacheDir, err)
	}

	repository := proj.Spec.Repository
	if repository == "" {
		repository = fmt.Sprintf("%s/%s", defaultGitHubOrganization, proj.Name)
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
