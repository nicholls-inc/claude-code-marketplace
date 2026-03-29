package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

// GitHubTask defines a label-based task for the GitHub source.
type GitHubTask struct {
	Labels []string
	Workflow string
}

// GitHub scans GitHub issues and produces vessels.
type GitHub struct {
	Repo      string
	Tasks     map[string]GitHubTask
	Exclude   []string
	Queue     *queue.Queue
	CmdRunner CommandRunner
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (g *GitHub) Name() string { return "github-issue" }

func (g *GitHub) Scan(ctx context.Context) ([]queue.Vessel, error) {
	excludeSet := make(map[string]bool, len(g.Exclude))
	for _, ex := range g.Exclude {
		excludeSet[ex] = true
	}

	var vessels []queue.Vessel
	for _, task := range g.Tasks {
		args := []string{
			"search", "issues",
			"--repo", g.Repo,
			"--state", "open",
			"--json", "number,title,url,labels",
			"--limit", "20",
		}
		for _, label := range task.Labels {
			args = append(args, "--label", label)
		}

		out, err := g.CmdRunner.Run(ctx, "gh", args...)
		if err != nil {
			return vessels, fmt.Errorf("gh search issues: %w", err)
		}

		var issues []ghIssue
		if err := json.Unmarshal(out, &issues); err != nil {
			return vessels, fmt.Errorf("parse gh search output: %w", err)
		}

		for _, issue := range issues {
			if g.hasExcludedLabel(issue, excludeSet) ||
				g.Queue.HasRef(issue.URL) ||
				g.hasBranch(ctx, issue.Number) ||
				g.hasOpenPR(ctx, issue.Number) {
				continue
			}

			vessels = append(vessels, queue.Vessel{
				ID:     fmt.Sprintf("issue-%d", issue.Number),
				Source: "github-issue",
				Ref:    issue.URL,
				Workflow: task.Workflow,
				Meta: map[string]string{
					"issue_num": strconv.Itoa(issue.Number),
				},
				State:     queue.StatePending,
				CreatedAt: time.Now().UTC(),
			})
		}
	}
	return vessels, nil
}

func (g *GitHub) OnStart(ctx context.Context, vessel queue.Vessel) error {
	if g.CmdRunner == nil {
		return nil
	}
	issueNum := vessel.Meta["issue_num"]
	if issueNum == "" {
		return nil
	}
	// Best-effort: add in-progress label
	_, _ = g.CmdRunner.Run(ctx, "gh", "issue", "edit",
		issueNum,
		"--repo", g.Repo,
		"--add-label", "in-progress")
	return nil
}

func (g *GitHub) BranchName(vessel queue.Vessel) string {
	prefix := "feat"
	if strings.Contains(strings.ToLower(vessel.Workflow), "fix") {
		prefix = "fix"
	}
	issueNum := vessel.Meta["issue_num"]
	slug := slugify(vessel.Ref)
	return fmt.Sprintf("%s/issue-%s-%s", prefix, issueNum, slug)
}

func (g *GitHub) hasExcludedLabel(issue ghIssue, excluded map[string]bool) bool {
	for _, l := range issue.Labels {
		if excluded[l.Name] {
			return true
		}
	}
	return false
}

// branchPrefixes lists the branch name prefixes xylem uses when creating
// worktree branches.
var branchPrefixes = []string{"fix", "feat"}

func (g *GitHub) hasBranch(ctx context.Context, issueNum int) bool {
	for _, prefix := range branchPrefixes {
		pattern := fmt.Sprintf("%s/issue-%d-*", prefix, issueNum)
		out, err := g.CmdRunner.Run(ctx, "git", "ls-remote", "--heads", "origin", pattern)
		if err == nil && strings.Contains(string(out), "\t") {
			return true
		}
	}
	return false
}

func (g *GitHub) hasOpenPR(ctx context.Context, issueNum int) bool {
	for _, prefix := range branchPrefixes {
		search := fmt.Sprintf("head:%s/issue-%d-", prefix, issueNum)
		out, err := g.CmdRunner.Run(ctx, "gh", "pr", "list",
			"--repo", g.Repo,
			"--search", search,
			"--state", "open",
			"--json", "number,headRefName",
			"--limit", "5")
		if err != nil {
			continue
		}
		var prs []struct {
			Number      int    `json:"number"`
			HeadRefName string `json:"headRefName"`
		}
		if err := json.Unmarshal(out, &prs); err != nil {
			continue
		}
		branchPrefix := fmt.Sprintf("%s/issue-%d-", prefix, issueNum)
		for _, pr := range prs {
			if strings.HasPrefix(pr.HeadRefName, branchPrefix) {
				return true
			}
		}
	}
	return false
}
