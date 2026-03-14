package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

// CommandRunner abstracts shell calls for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Scanner scans GitHub for actionable issues and enqueues vessels.
type Scanner struct {
	Config    *config.Config
	Queue     *queue.Queue
	CmdRunner CommandRunner
}

// ScanResult summarises a scan run.
type ScanResult struct {
	Added   int
	Skipped int
	Paused  bool
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// New creates a Scanner.
func New(cfg *config.Config, q *queue.Queue, runner CommandRunner) *Scanner {
	return &Scanner{Config: cfg, Queue: q, CmdRunner: runner}
}

// Scan queries GitHub, filters issues, and enqueues new vessels.
func (s *Scanner) Scan(ctx context.Context) (ScanResult, error) {
	pauseMarker := filepath.Join(s.Config.StateDir, "paused")
	if _, err := os.Stat(pauseMarker); err == nil {
		return ScanResult{Paused: true}, nil
	}

	var result ScanResult
	excludeSet := make(map[string]bool, len(s.Config.Exclude))
	for _, ex := range s.Config.Exclude {
		excludeSet[ex] = true
	}

	for _, task := range s.Config.Tasks {
		args := []string{
			"search", "issues",
			"--repo", s.Config.Repo,
			"--state", "open",
			"--json", "number,title,url,labels",
			"--limit", "20",
		}
		for _, label := range task.Labels {
			args = append(args, "--label", label)
		}

		out, err := s.CmdRunner.Run(ctx, "gh", args...)
		if err != nil {
			return result, fmt.Errorf("gh search issues: %w", err)
		}

		var issues []ghIssue
		if err := json.Unmarshal(out, &issues); err != nil {
			return result, fmt.Errorf("parse gh search output: %w", err)
		}

		for _, issue := range issues {
			if s.hasExcludedLabel(issue, excludeSet) ||
				s.Queue.HasIssue(issue.Number) ||
				s.hasBranch(ctx, issue.Number) ||
				s.hasOpenPR(ctx, issue.Number) {
				result.Skipped++
				continue
			}

			vessel := queue.Vessel{
				ID:        fmt.Sprintf("issue-%d", issue.Number),
				IssueURL:  issue.URL,
				IssueNum:  issue.Number,
				Skill:     task.Skill,
				State:     queue.StatePending,
				CreatedAt: time.Now().UTC(),
			}
			if err := s.Queue.Enqueue(vessel); err != nil {
				return result, fmt.Errorf("enqueue issue %d: %w", issue.Number, err)
			}
			result.Added++
		}
	}
	return result, nil
}

func (s *Scanner) hasExcludedLabel(issue ghIssue, excluded map[string]bool) bool {
	for _, l := range issue.Labels {
		if excluded[l.Name] {
			return true
		}
	}
	return false
}

func (s *Scanner) hasBranch(ctx context.Context, issueNum int) bool {
	for _, prefix := range branchPrefixes {
		pattern := fmt.Sprintf("%s/issue-%d-*", prefix, issueNum)
		out, err := s.CmdRunner.Run(ctx, "git", "ls-remote", "--heads", "origin", pattern)
		// git ls-remote output is "<hash>\t<refname>" — must contain a tab to be valid
		if err == nil && strings.Contains(string(out), "\t") {
			return true
		}
	}
	return false
}

// branchPrefixes lists the branch name prefixes xylem uses when creating
// worktree branches. Both hasBranch and hasOpenPR use this list so they stay
// in sync.
var branchPrefixes = []string{"fix", "feat"}

func (s *Scanner) hasOpenPR(ctx context.Context, issueNum int) bool {
	// Search for PRs whose head branch matches xylem's naming convention.
	// Using "head:" qualifier limits the search to the branch name rather than
	// matching "#N" anywhere in the PR title/body (which caused false positives).
	for _, prefix := range branchPrefixes {
		search := fmt.Sprintf("head:%s/issue-%d-", prefix, issueNum)
		out, err := s.CmdRunner.Run(ctx, "gh", "pr", "list",
			"--repo", s.Config.Repo,
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
		// Verify the head branch actually matches the expected pattern to guard
		// against search-API approximations.
		branchPrefix := fmt.Sprintf("%s/issue-%d-", prefix, issueNum)
		for _, pr := range prs {
			if strings.HasPrefix(pr.HeadRefName, branchPrefix) {
				return true
			}
		}
	}
	return false
}
