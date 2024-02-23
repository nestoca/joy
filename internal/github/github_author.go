package github

import (
	"fmt"

	"github.com/nestoca/joy/api/v1alpha1"
)

func GetCommitGitHubAuthor(project *v1alpha1.Project, defaultOrganization, commitSHA string) (string, error) {
	repository := project.Spec.Repository
	if repository == "" {
		repository = fmt.Sprintf("%s/%s", defaultOrganization, project.Name)
	}

	out, err := executeAndGetOutput(".", "api", "--jq", ".author.login", "repos/"+repository+"/commits/"+commitSHA)
	if err != nil {
		return "", fmt.Errorf("getting commit's GitHub author: %w", err)
	}
	return out, nil
}
