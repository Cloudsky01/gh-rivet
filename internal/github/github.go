package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

const DefaultTimeout = 30 * time.Second

type Client struct {
	repo    string
	timeout time.Duration
}

func NewClient(repo string) *Client {
	return NewClientWithTimeout(repo, DefaultTimeout)
}

func NewClientWithTimeout(repo string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{
		repo:    repo,
		timeout: timeout,
	}
}

func (c *Client) GetRepository() string {
	return c.repo
}

func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

func (c *Client) GetLatestRun() (*models.GHRun, error) {
	runs, err := c.GetRecentRuns(1)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("no workflow runs found")
	}
	return &runs[0], nil
}

func (c *Client) GetRecentRuns(limit int) ([]models.GHRun, error) {
	return c.GetWorkflowRuns("", limit)
}

func (c *Client) GetWorkflowRuns(workflowName string, limit int) ([]models.GHRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{"run", "list", "--limit", fmt.Sprintf("%d", limit), "--json", "databaseId,displayTitle,workflowName,status,conclusion,createdAt,headBranch"}

	if workflowName != "" {
		args = append(args, "--workflow", workflowName)
	}

	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gh run list timed out after %v", c.timeout)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("gh run list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh run list failed: %w", err)
	}

	var runs []models.GHRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CreatedAt.Equal(runs[j].CreatedAt) {
			return runs[i].DatabaseID > runs[j].DatabaseID
		}
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	return runs, nil
}

func (c *Client) GetRunByID(runID int) (*models.GHRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{"run", "view", fmt.Sprintf("%d", runID), "--json", "databaseId,displayTitle,workflowName,status,conclusion,createdAt,headBranch"}

	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gh run view timed out after %v", c.timeout)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("gh run view failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh run view failed: %w", err)
	}

	var run models.GHRun
	if err := json.Unmarshal(output, &run); err != nil {
		return nil, fmt.Errorf("failed to parse workflow run: %w", err)
	}

	return &run, nil
}

func (c *Client) GetRunJobs(runID int) ([]models.GHJob, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{"run", "view", fmt.Sprintf("%d", runID), "--json", "jobs"}

	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gh run view timed out after %v", c.timeout)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("gh run view failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh run view failed: %w", err)
	}

	var detail models.GHRunDetail
	if err := json.Unmarshal(output, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse job details: %w", err)
	}

	return detail.Jobs, nil
}

func (c *Client) GetJobsFromRuns(runs []models.GHRun) ([]models.GHJob, error) {
	var allJobs []models.GHJob

	for _, run := range runs {
		jobs, err := c.GetRunJobs(run.DatabaseID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get jobs for run %d (%s): %v\n",
				run.DatabaseID, run.WorkflowName, err)
			continue
		}

		for i := range jobs {
			jobs[i].WorkflowName = run.WorkflowName
			jobs[i].RunID = run.DatabaseID
		}

		allJobs = append(allJobs, jobs...)
	}

	return allJobs, nil
}

func (c *Client) OpenWorkflowInBrowser(workflowName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{"workflow", "view", workflowName, "-w"}

	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gh workflow view timed out after %v", c.timeout)
		}
		return fmt.Errorf("failed to open workflow in browser: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func (c *Client) OpenRunInBrowser(runID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{"run", "view", fmt.Sprintf("%d", runID), "-w"}

	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gh run view timed out after %v", c.timeout)
		}
		return fmt.Errorf("failed to open run in browser: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// RepositoryExists checks if a repository exists on GitHub
func (c *Client) RepositoryExists(ctx context.Context, repo string) (bool, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Use gh api to check if repository exists
	args := []string{"api", fmt.Sprintf("repos/%s", repo)}

	cmd := exec.CommandContext(cmdCtx, "gh", args...)
	output, err := cmd.Output()

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return false, fmt.Errorf("gh api timed out after %v", c.timeout)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Repository doesn't exist or is not accessible
			stderr := string(exitErr.Stderr)
			if stderr != "" {
				return false, fmt.Errorf("repository not found or not accessible: %s", stderr)
			}
		}
		return false, fmt.Errorf("gh api failed: %w", err)
	}

	// If we get valid JSON response, repository exists
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return true, nil
}

// GetWorkflows fetches the list of workflow files from a repository
func (c *Client) GetWorkflows(ctx context.Context, repo string) ([]string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"api", "--paginate", fmt.Sprintf("repos/%s/actions/workflows", repo), "--jq", ".workflows[].path"}
	cmd := exec.CommandContext(cmdCtx, "gh", args...)
	output, err := cmd.Output()

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gh api timed out after %v", c.timeout)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to fetch workflows: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}

	return parseWorkflowPaths(string(output)), nil
}

func parseWorkflowPaths(output string) []string {
	var workflows []string
	const prefix = ".github/workflows/"

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			workflows = append(workflows, line[len(prefix):])
		}
	}

	sort.Strings(workflows)
	return workflows
}
