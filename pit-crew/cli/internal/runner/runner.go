package runner

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

// CommandRunner abstracts subprocess execution for testing.
type CommandRunner interface {
	RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
	RunProcess(ctx context.Context, dir string, name string, args ...string) error
}

// WorktreeManager abstracts worktree lifecycle for testing.
type WorktreeManager interface {
	Create(ctx context.Context, branchName string) (string, error)
}

// DrainResult summarises a drain run.
type DrainResult struct {
	Completed int
	Failed    int
	Skipped   int
}

// Runner launches Claude sessions for queued jobs with concurrency control.
type Runner struct {
	Config   *config.Config
	Queue    *queue.Queue
	Worktree WorktreeManager
	Runner   CommandRunner
}

// New creates a Runner.
func New(cfg *config.Config, q *queue.Queue, wt WorktreeManager, r CommandRunner) *Runner {
	return &Runner{Config: cfg, Queue: q, Worktree: wt, Runner: r}
}

// Drain dequeues pending jobs and launches sessions up to Config.Concurrency concurrently.
// On context cancellation, no new jobs are launched; running jobs complete normally.
func (r *Runner) Drain(ctx context.Context) (DrainResult, error) {
	timeout, err := time.ParseDuration(r.Config.Timeout)
	if err != nil {
		return DrainResult{}, fmt.Errorf("parse timeout: %w", err)
	}

	sem := make(chan struct{}, r.Config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var result DrainResult

	for {
		select {
		case <-ctx.Done():
			goto wait
		default:
		}

		job, err := r.Queue.Dequeue()
		if err != nil || job == nil {
			break
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(j queue.Job) {
			defer wg.Done()
			defer func() { <-sem }()

			jobCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			outcome := r.runJob(jobCtx, j)

			mu.Lock()
			switch outcome {
			case "completed":
				result.Completed++
			case "failed":
				result.Failed++
			default:
				result.Skipped++
			}
			mu.Unlock()
		}(*job)
	}

wait:
	wg.Wait()
	return result, nil
}

func (r *Runner) runJob(ctx context.Context, job queue.Job) string {
	// Add in-progress label (best-effort)
	r.Runner.RunOutput(ctx, "gh", "issue", "edit", //nolint:errcheck
		fmt.Sprintf("%d", job.IssueNum),
		"--repo", r.Config.Repo,
		"--add-label", "in-progress")

	// Determine branch name
	prefix := "feat"
	if strings.Contains(strings.ToLower(job.Skill), "fix") {
		prefix = "fix"
	}
	slug := slugify(job.IssueURL)
	branchName := fmt.Sprintf("%s/issue-%d-%s", prefix, job.IssueNum, slug)

	// Create worktree
	worktreePath, err := r.Worktree.Create(ctx, branchName)
	if err != nil {
		r.Queue.Update(job.ID, queue.StateFailed, fmt.Sprintf("create worktree: %v", err)) //nolint:errcheck
		return "failed"
	}

	// Build and launch claude command
	cmd, args, err := buildCommand(r.Config, &job)
	if err != nil {
		r.Queue.Update(job.ID, queue.StateFailed, fmt.Sprintf("build command: %v", err)) //nolint:errcheck
		return "failed"
	}

	runErr := r.Runner.RunProcess(ctx, worktreePath, cmd, args...)

	if runErr != nil {
		r.Queue.Update(job.ID, queue.StateFailed, runErr.Error()) //nolint:errcheck
		return "failed"
	}

	r.Queue.Update(job.ID, queue.StateCompleted, "") //nolint:errcheck
	return "completed"
}

// cmdVars holds template substitution values.
type cmdVars struct {
	Command  string
	Skill    string
	IssueURL string
	MaxTurns int
}

// buildCommand renders the config template and returns (cmd, args).
func buildCommand(cfg *config.Config, job *queue.Job) (string, []string, error) {
	tmpl, err := template.New("cmd").Parse(cfg.Claude.Template)
	if err != nil {
		return "", nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cmdVars{
		Command:  cfg.Claude.Command,
		Skill:    job.Skill,
		IssueURL: job.IssueURL,
		MaxTurns: cfg.MaxTurns,
	}); err != nil {
		return "", nil, fmt.Errorf("execute template: %w", err)
	}
	parts := strings.Fields(buf.String())
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command from template")
	}
	return parts[0], parts[1:], nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify produces a short kebab-case slug from a string (uses last path segment for URLs).
func slugify(s string) string {
	// For URLs, use the last segment (issue number already in branch name)
	// Just return a short fixed slug so branch names stay clean
	parts := strings.Split(strings.ToLower(s), "/")
	src := parts[len(parts)-1]
	if src == "" && len(parts) > 1 {
		src = parts[len(parts)-2]
	}
	clean := nonAlphaNum.ReplaceAllString(src, "-")
	clean = strings.Trim(clean, "-")
	if len(clean) > 20 {
		clean = clean[:20]
	}
	if clean == "" {
		clean = "task"
	}
	return clean
}
