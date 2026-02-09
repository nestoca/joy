package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/nestoca/joy/internal/git/pr"
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

type label struct {
	Name string `json:"name"`
}

type pullRequest struct {
	HeadRefName string  `json:"headRefName"`
	Labels      []label `json:"labels"`
}

func (p *PullRequestProvider) EnsureInstalledAndAuthenticated() error {
	return EnsureInstalledAndAuthenticated()
}

func (p *PullRequestProvider) Exists(branch string) (bool, error) {
	pr, err := p.get(branch)
	if err != nil {
		return false, fmt.Errorf("getting pull request for branch %s: %w", branch, err)
	}
	return pr != nil, nil
}

func (p *PullRequestProvider) GetBranchesPromotingToEnvironment(env string) ([]string, error) {
	prs, err := p.getAllWithLabel(fmt.Sprintf("promote:%s", env))
	if err != nil {
		return nil, fmt.Errorf("getting pull requests: %w", err)
	}

	var branches []string
	for _, pr := range prs {
		branches = append(branches, pr.HeadRefName)
	}
	return branches, nil
}

func (p *PullRequestProvider) CreateInteractively(branch string) error {
	err := executeInteractively(p.dir, "pr", "create", "--head", branch)
	if err != nil {
		return fmt.Errorf("creating pull request for branch %s: %w", branch, err)
	}
	return nil
}

func (p *PullRequestProvider) Create(params pr.CreateParams) (string, error) {
	args := []string{
		"pr", "create",
		"--head", params.Branch,
		"--title", params.Title,
		"--body", params.Body,
	}

	if params.Draft {
		args = append(args, "--draft")
	}

	for _, label := range params.Labels {
		args = append(args, "--label", label)
	}

	for _, reviewer := range params.Reviewers {
		// Bots cannot be requested as reviewers (GitHub returns "not found")
		if strings.HasSuffix(reviewer, "[bot]") || reviewer == "nestobot" {
			continue
		}
		args = append(args, "--reviewer", reviewer)
	}

	err := p.createLabels(params.Labels...)
	if err != nil {
		return "", err
	}
	prURL, err := ExecuteAndGetOutput(p.dir, args...)
	if err != nil {
		return "", fmt.Errorf("creating pull request for branch %s: %w", params.Branch, err)
	}

	return strings.TrimSpace(prURL), err
}

func (p *PullRequestProvider) getPromotionLabels(branch string) ([]string, error) {
	pr, err := p.get(branch)
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

func (p *PullRequestProvider) GetPromotionEnvironment(branch string) (string, error) {
	labels, err := p.getPromotionLabels(branch)
	if err != nil {
		return "", fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}
	if len(labels) == 0 {
		return "", nil
	}
	return labelRegex.FindStringSubmatch(labels[0])[1], nil
}

func (p *PullRequestProvider) SetPromotionEnvironment(branch string, env string) error {
	// Get current promotion labels
	// Typically, there is only one or none, but we cannot guarantee there are not many
	labels, err := p.getPromotionLabels(branch)
	if err != nil {
		return fmt.Errorf("getting promotion labels for branch %s: %w", branch, err)
	}

	// Remove existing labels, if any
	for _, label := range labels {
		if err := p.removeLabel(branch, label); err != nil {
			return fmt.Errorf("removing label %s from branch %s: %w", label, branch, err)
		}
	}

	// Add new label
	if env != "" {
		label := fmt.Sprintf("promote:%s", env)
		if err := p.addLabel(branch, label); err != nil {
			return fmt.Errorf("adding label %s to branch %s: %w", env, branch, err)
		}
	}
	return nil
}

func (p *PullRequestProvider) get(branch string) (*pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--head", branch, "--state", "open", "--json", "headRefName,labels")
	cmd.Dir = p.dir
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

func (p *PullRequestProvider) getAllWithLabel(label string) ([]pullRequest, error) {
	// List pull requests for branch
	cmd := exec.Command("gh", "pr", "list", "--label", label, "--state", "open", "--json", "headRefName")
	cmd.Dir = p.dir
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

func (p *PullRequestProvider) removeLabel(branch string, label string) error {
	cmd := exec.Command("gh", "pr", "edit", branch, "--remove-label", label)
	cmd.Dir = p.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing label %s from branch %s: %s", label, branch, output)
	}
	return nil
}

func (p *PullRequestProvider) createLabels(labels ...string) error {
	if len(labels) == 0 {
		return nil
	}
	cmd := exec.Command("gh", "label", "list", "--json", "name", "--limit", "1000")
	cmd.Dir = p.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("listing labels: %s", output)
	}

	var existingLabels []label
	if err := json.Unmarshal(output, &existingLabels); err != nil {
		return fmt.Errorf("unmarshaling label list: %w", err)
	}
	existingLabelSet := make(map[string]bool)
	for _, existingLabel := range existingLabels {
		existingLabelSet[existingLabel.Name] = true
	}
	for _, label := range labels {
		if !existingLabelSet[label] {
			cmd = exec.Command("gh", "label", "create", label)
			cmd.Dir = p.dir
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("creating label %s: %s", label, output)
			}
		}
	}
	return nil
}

func (p *PullRequestProvider) addLabel(branch string, label string) error {
	// Ensure label exists in repo
	err := p.createLabels(label)
	if err != nil {
		return err
	}

	// Add label to PR
	cmd := exec.Command("gh", "pr", "edit", branch, "--add-label", label)
	cmd.Dir = p.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adding label %s to branch %s: %s", label, branch, output)
	}
	return nil
}
