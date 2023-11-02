package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type PullRequestProvider struct {
	dir string
}

func NewPullRequestProvider(dir string) *PullRequestProvider {
	return &PullRequestProvider{
		dir: dir,
	}
}

var labelRegex = regexp.MustCompile(`^promote:(.+)$`)

type pullRequest struct {
	HeadRefName string `json:"headRefName"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (g *PullRequestProvider) EnsureInstalledAndAuthenticated() error {
	return EnsureInstalledAndAuthenticated()
}

func (g *PullRequestProvider) Exists(branch string) (bool, error) {
	pr, err := g.get(branch)
	if err != nil {
		return false, fmt.Errorf("getting pull request for branch %s: %w", branch, err)
	}
	return pr != nil, nil
}

func (g *PullRequestProvider) GetBranchesPromotingToEnvironment(env string) ([]string, error) {
	prs, err := g.getAllWithLabel(fmt.Sprintf("promote:%s", env))
	if err != nil {
		return nil, fmt.Errorf("getting pull requests: %w", err)
	}

	var branches []string
	for _, pr := range prs {
		branches = append(branches, pr.HeadRefName)
	}
	return branches, nil
}

func (g *PullRequestProvider) CreateInteractively(branch string) error {
	err := executeInteractively(g.dir, "pr", "create", "--head", branch)
	if err != nil {
		return fmt.Errorf("creating pull request for branch %s: %w", branch, err)
	}
	return nil
}

func (g *PullRequestProvider) Create(branch, title, body string) (string, error) {
	prURL, err := executeAndGetOutput(g.dir, "pr", "create", "--head", branch, "--title", title, "--body", body)
	if err != nil {
		return "", fmt.Errorf("creating pull request for branch %s: %w", branch, err)
	}
	prURL = strings.TrimSpace(prURL)
	return prURL, err
}

func (g *PullRequestProvider) getPromotionLabels(branch string) ([]string, error) {
	pr, err := g.get(branch)
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

func (g *PullRequestProvider) GetPromotionEnvironment(branch string) (string, error) {
	labels, err := g.getPromotionLabels(branch)
	if err != nil {
		return "", fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}
	if len(labels) == 0 {
		return "", nil
	}
	return labelRegex.FindStringSubmatch(labels[0])[1], nil
}

func (g *PullRequestProvider) SetPromotionEnvironment(branch string, env string) error {
	// Get current promotion labels
	// Typically, there is only one or none, but we cannot guarantee there are not many
	labels, err := g.getPromotionLabels(branch)
	if err != nil {
		return fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}

	// Remove existing labels, if any
	for _, label := range labels {
		if err := g.removeLabel(branch, label); err != nil {
			return fmt.Errorf("removing label %s from branch %s: %w", label, branch, err)
		}
	}

	// Add new label
	if env != "" {
		label := fmt.Sprintf("promote:%s", env)
		if err := g.addLabel(branch, label); err != nil {
			return fmt.Errorf("adding label %s to branch %s: %w", env, branch, err)
		}
	}
	return nil
}

func (g *PullRequestProvider) get(branch string) (*pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--head", branch, "--state", "open", "--json", "headRefName,labels")
	cmd.Dir = g.dir
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

func (g *PullRequestProvider) getAllWithLabel(label string) ([]pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--label", label, "--state", "open", "--json", "headRefName")
	cmd.Dir = g.dir
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

func (g *PullRequestProvider) removeLabel(branch string, label string) error {
	cmd := exec.Command("gh", "pr", "edit", branch, "--remove-label", label)
	cmd.Dir = g.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing label %s from branch %s: %s", label, branch, output)
	}
	return nil
}

func (g *PullRequestProvider) addLabel(branch string, label string) error {
	// Ensure label exists in repo
	cmd := exec.Command("gh", "label", "create", label)
	cmd.Dir = g.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(output), "already exists") {
			return fmt.Errorf("creating label %s: %s", label, output)
		}
	}

	// Add label to PR
	cmd = exec.Command("gh", "pr", "edit", branch, "--add-label", label)
	cmd.Dir = g.dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adding label %s to branch %s: %s", label, branch, output)
	}
	return nil
}
