package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/nestoca/joy/internal/git"
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
		"--body", truncateBody(params.Body),
	}

	if params.Draft {
		args = append(args, "--draft")
	}

	for _, label := range params.Labels {
		args = append(args, "--label", label)
	}

	for _, reviewer := range requestableReviewers(params.Reviewers) {
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

const pullRequestFields = "number,url,headRefName,headRefOid,isCrossRepository,body,labels,state,reviewDecision,commits"

type pullRequestDetails struct {
	Number         int      `json:"number"`
	URL            string   `json:"url"`
	HeadRefName    string   `json:"headRefName"`
	HeadRefOid     string   `json:"headRefOid"`
	IsCrossRepo    bool     `json:"isCrossRepository"`
	Body           string   `json:"body"`
	Labels         []label  `json:"labels"`
	State          string   `json:"state"`
	ReviewDecision string   `json:"reviewDecision"`
	Commits        []commit `json:"commits"`
}

type commit struct {
	OID string `json:"oid"`
}

func (d pullRequestDetails) toPullRequest() pr.PullRequest {
	result := pr.PullRequest{
		Number:          d.Number,
		URL:             d.URL,
		Branch:          d.HeadRefName,
		Body:            d.Body,
		Open:            d.State == "OPEN",
		Approved:        d.ReviewDecision == "APPROVED",
		HeadSHA:         d.HeadRefOid,
		CrossRepository: d.IsCrossRepo,
	}
	for _, label := range d.Labels {
		result.Labels = append(result.Labels, label.Name)
	}
	for _, commit := range d.Commits {
		result.Commits = append(result.Commits, commit.OID)
	}
	return result
}

func (p *PullRequestProvider) Get(pullRequest string) (*pr.PullRequest, error) {
	output, err := ExecuteAndGetOutput(p.dir, "pr", "view", pullRequest, "--json", pullRequestFields)
	if err != nil {
		return nil, fmt.Errorf("getting pull request %s: %w", pullRequest, err)
	}

	var details pullRequestDetails
	if err := json.Unmarshal([]byte(output), &details); err != nil {
		return nil, fmt.Errorf("unmarshaling pull request %s: %w", pullRequest, err)
	}

	repository, err := repositoryFromPullRequestURL(details.URL)
	if err != nil {
		return nil, err
	}
	catalogRepository, err := ExecuteAndGetOutput(p.dir, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return nil, fmt.Errorf("getting repository: %w", err)
	}
	if !strings.EqualFold(repository, catalogRepository) {
		return nil, fmt.Errorf("pull request %s belongs to %s instead of %s", pullRequest, repository, catalogRepository)
	}

	result := details.toPullRequest()
	return &result, nil
}

// repositoryFromPullRequestURL returns the owner/name of the repository of given pull request URL, such as
// https://github.com/owner/name/pull/123.
func repositoryFromPullRequestURL(url string) (string, error) {
	path, ok := strings.CutPrefix(url, "https://github.com/")
	parts := strings.Split(path, "/")
	if !ok || len(parts) < 4 || parts[2] != "pull" {
		return "", fmt.Errorf("unexpected pull request url %q", url)
	}
	return parts[0] + "/" + parts[1], nil
}

func (p *PullRequestProvider) List(search string, limit int) ([]pr.PullRequest, error) {
	output, err := ExecuteAndGetOutput(p.dir, "pr", "list", "--state", "all", "--search", search, "--limit", strconv.Itoa(limit), "--json", pullRequestFields)
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}

	var details []pullRequestDetails
	if err := json.Unmarshal([]byte(output), &details); err != nil {
		return nil, fmt.Errorf("unmarshaling pull request list: %w", err)
	}

	var result []pr.PullRequest
	for _, d := range details {
		result = append(result, d.toPullRequest())
	}
	return result, nil
}

func (p *PullRequestProvider) FetchHead(number int) error {
	return git.FetchPullRequestHead(p.dir, number)
}

func (p *PullRequestProvider) Update(params pr.UpdateParams) (string, error) {
	args := []string{
		"pr", "edit", params.PullRequest,
		"--title", params.Title,
		"--body", truncateBody(params.Body),
	}

	for _, label := range params.Labels {
		args = append(args, "--add-label", label)
	}

	for _, reviewer := range requestableReviewers(params.Reviewers) {
		args = append(args, "--add-reviewer", reviewer)
	}

	if err := p.createLabels(params.Labels...); err != nil {
		return "", err
	}
	prURL, err := ExecuteAndGetOutput(p.dir, args...)
	if err != nil {
		return "", fmt.Errorf("updating pull request %s: %w", params.PullRequest, err)
	}

	// Labels such as rendering:failed are only removed once the update fully succeeded, so that a partially updated
	// pull request can still be found by them.
	if len(params.RemoveLabels) > 0 {
		args := []string{"pr", "edit", params.PullRequest}
		for _, label := range params.RemoveLabels {
			args = append(args, "--remove-label", label)
		}
		if _, err := ExecuteAndGetOutput(p.dir, args...); err != nil {
			return "", fmt.Errorf("removing labels from pull request %s: %w", params.PullRequest, err)
		}
	}

	return strings.TrimSpace(prURL), nil
}

// maxBodyLength is the maximum number of characters GitHub accepts in a pull request body.
const maxBodyLength = 65536

const truncatedBodyNotice = "\n\n⚠️ Truncated: the body exceeded GitHub's limit of 65536 characters."

// truncateBody returns given body cut at the last line break fitting GitHub's maximum pull request body length,
// followed by a notice, or the body itself when it fits.
func truncateBody(body string) string {
	runes := []rune(body)
	if len(runes) <= maxBodyLength {
		return body
	}

	truncated := string(runes[:maxBodyLength-len([]rune(truncatedBodyNotice))])
	if i := strings.LastIndex(truncated, "\n"); i > 0 {
		truncated = truncated[:i]
	}
	return truncated + truncatedBodyNotice
}

func (p *PullRequestProvider) AddLabels(pullRequest string, labels []string) error {
	if err := p.createLabels(labels...); err != nil {
		return err
	}
	args := []string{"pr", "edit", pullRequest}
	for _, label := range labels {
		args = append(args, "--add-label", label)
	}
	if _, err := ExecuteAndGetOutput(p.dir, args...); err != nil {
		return fmt.Errorf("adding labels to pull request %s: %w", pullRequest, err)
	}
	return nil
}

// requestableReviewers filters out bots, which cannot be requested as reviewers (GitHub returns "not found").
func requestableReviewers(reviewers []string) []string {
	var result []string
	for _, reviewer := range reviewers {
		if strings.HasSuffix(reviewer, "[bot]") || reviewer == "nestobot" {
			continue
		}
		result = append(result, reviewer)
	}
	return result
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

	// `gh label list --json` returns empty output when the repo has no labels,
	// which is not valid JSON and would fail to unmarshal.
	output = bytes.TrimSpace(output)
	var existingLabels []label
	if len(output) > 0 {
		if err := json.Unmarshal(output, &existingLabels); err != nil {
			return fmt.Errorf("unmarshaling label list: %w", err)
		}
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
