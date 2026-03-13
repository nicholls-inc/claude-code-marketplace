package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

// CommandRunner abstracts shell calls for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Scanner scans GitHub for actionable issues and enqueues jobs.
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

// Scan queries GitHub, filters issues, and enqueues new jobs.
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

			job := queue.Job{
				ID:        fmt.Sprintf("issue-%d", issue.Number),
				IssueURL:  issue.URL,
				IssueNum:  issue.Number,
				Skill:     task.Skill,
				State:     queue.StatePending,
				CreatedAt: time.Now().UTC(),
			}
			if err := s.Queue.Enqueue(job); err != nil {
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
	for _, prefix := range []string{"fix", "feat"} {
		pattern := fmt.Sprintf("%s/issue-%d-*", prefix, issueNum)
		out, err := s.CmdRunner.Run(ctx, "git", "ls-remote", "--heads", "origin", pattern)
		// git ls-remote output is "<hash>\t<refname>" — must contain a tab to be valid
		if err == nil && strings.Contains(string(out), "\t") {
			return true
		}
	}
	return false
}

func (s *Scanner) hasOpenPR(ctx context.Context, issueNum int) bool {
	search := fmt.Sprintf("#%d", issueNum)
	out, err := s.CmdRunner.Run(ctx, "gh", "pr", "list",
		"--repo", s.Config.Repo,
		"--search", search,
		"--state", "open",
		"--json", "number",
		"--limit", "1")
	if err != nil {
		return false
	}
	var prs []struct{ Number int }
	if err := json.Unmarshal(out, &prs); err != nil {
		return false
	}
	return len(prs) > 0
}
