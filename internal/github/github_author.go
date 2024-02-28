package github

import (
	"encoding/json"
	"fmt"

	"github.com/nestoca/joy/api/v1alpha1"
)

type commitInfo struct {
	Sha   string `json:"sha"`
	Login string `json:"login"`
}

func GetCommitsGitHubAuthors(project *v1alpha1.Project, defaultOrganization, fromTag, toTag string) (map[string]string, error) {
	repository := project.Spec.Repository
	if repository == "" {
		repository = fmt.Sprintf("%s/%s", defaultOrganization, project.Name)
	}

	path := fmt.Sprintf("repos/%s/compare/%s...%s", repository, fromTag, toTag)
	data, err := executeAndGetOutput(".", "api", "--jq", "[.commits[] | {sha: .sha, login: (.author.login // .committer.login)}]", path)
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
