package promote

import (
	"encoding/json"
	"fmt"
	"github.com/nestoca/joy/internal/gh"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type GitHubPullRequestProvider struct {
}

var labelRegex = regexp.MustCompile(`^promote:\s*(\S+)$`)

type pullRequest struct {
	HeadRefName string `json:"headRefName"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (g *GitHubPullRequestProvider) EnsureInstalledAndAuthorized() error {
	return gh.EnsureInstalledAndAuthorized()
}

func (g *GitHubPullRequestProvider) Exists(branch string) (bool, error) {
	pr, err := get(branch)
	if err != nil {
		return false, fmt.Errorf("getting pull request for branch %s: %w", branch, err)
	}
	return pr != nil, nil
}

func (g *GitHubPullRequestProvider) GetBranchesPromotingToEnvironment(env string) ([]string, error) {
	prs, err := getAll()
	if err != nil {
		return nil, fmt.Errorf("getting pull requests: %w", err)
	}

	var branches []string
	for _, pr := range prs {
		for _, label := range pr.Labels {
			if labelRegex.MatchString(label.Name) && labelRegex.FindStringSubmatch(label.Name)[1] == env {
				branches = append(branches, pr.HeadRefName)
				break
			}
		}
	}
	return branches, nil
}

func (g *GitHubPullRequestProvider) CreateInteractively(branch string) error {
	cmd := exec.Command("gh", "pr", "create", "--head", branch)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}
	return nil
}

func getPromotionLabels(branch string) ([]string, error) {
	pr, err := get(branch)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, nil
	}
	var labels []string
	for _, label := range pr.Labels {
		if labelRegex.MatchString(label.Name) {
			labels = append(labels, label.Name)
		}
	}
	return labels, nil
}

func (g *GitHubPullRequestProvider) GetPromotionEnvironment(branch string) (string, error) {
	labels, err := getPromotionLabels(branch)
	if err != nil {
		return "", fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}
	if len(labels) == 0 {
		return "", nil
	}
	return labelRegex.FindStringSubmatch(labels[0])[1], nil
}

func (g *GitHubPullRequestProvider) SetPromotionEnvironment(branch string, env string) error {
	// Get current promotion labels
	// Typically, there is only one or none, but we cannot guarantee there are not many
	labels, err := getPromotionLabels(branch)
	if err != nil {
		return fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}

	// Remove existing labels, if any
	for _, label := range labels {
		if err := removeLabel(branch, label); err != nil {
			return fmt.Errorf("removing label %s from branch %s: %w", label, branch, err)
		}
	}

	// Add new label
	if env != "" {
		label := fmt.Sprintf("promote:%s", env)
		if err := addLabel(branch, label); err != nil {
			return fmt.Errorf("adding label %s to branch %s: %w", env, branch, err)
		}
	}
	return nil
}

func get(branch string) (*pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--head", branch, "--json", "headRefName,labels")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing pull requests for branch %s: %s", branch, output)
	}

	// Unmarshal JSON
	var prs []pullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("unmarshaling pull request list: %w", err)
	}

	// We can safely assume that there is either none or only one PR for a given branch
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

func getAll() ([]pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--json", "headRefName,labels")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %s", output)
	}

	// Unmarshal JSON
	var prs []pullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("unmarshaling pull request list: %w", err)
	}

	return prs, nil
}

func removeLabel(branch string, label string) error {
	cmd := exec.Command("gh", "pr", "edit", branch, "--remove-label", label)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing label %s from branch %s: %s", label, branch, output)
	}
	return nil
}

func addLabel(branch string, label string) error {
	// Ensure label exists in repo
	cmd := exec.Command("gh", "label", "create", label)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(output), "already exists") {
			return fmt.Errorf("creating label %s: %s", label, output)
		}
	}

	// Add label to PR
	cmd = exec.Command("gh", "pr", "edit", branch, "--add-label", label)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adding label %s to branch %s: %s", label, branch, output)
	}
	return nil
}
