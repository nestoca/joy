package promote

import (
	"fmt"
	"os/exec"
	"strings"
)

type CommitMetadata struct {
	Sha     string
	Author  string
	Message string
}

func GetCommitsMetadata(dir, from, to string) ([]*CommitMetadata, error) {
	gitArgs := []string{"log", "--pretty=format:%H%n%an%n%s", from + ".." + to}
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
