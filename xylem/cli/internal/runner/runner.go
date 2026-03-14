package runner

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
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

// Runner launches Claude sessions for queued vessels with concurrency control.
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

// Drain dequeues pending vessels and launches sessions up to Config.Concurrency concurrently.
// On context cancellation, no new vessels are launched; running vessels complete normally.
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

		vessel, err := r.Queue.Dequeue()
		if err != nil || vessel == nil {
			break
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(j queue.Vessel) {
			defer wg.Done()
			defer func() { <-sem }()

			vesselCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			outcome := r.runVessel(vesselCtx, j)

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
		}(*vessel)
	}

wait:
	wg.Wait()
	return result, nil
}

func (r *Runner) runVessel(ctx context.Context, vessel queue.Vessel) string {
	// Add in-progress label (best-effort)
	r.Runner.RunOutput(ctx, "gh", "issue", "edit", //nolint:errcheck
		fmt.Sprintf("%d", vessel.IssueNum),
		"--repo", r.Config.Repo,
		"--add-label", "in-progress")

	// Determine branch name
	prefix := "feat"
	if strings.Contains(strings.ToLower(vessel.Skill), "fix") {
		prefix = "fix"
	}
	slug := slugify(vessel.IssueURL)
	branchName := fmt.Sprintf("%s/issue-%d-%s", prefix, vessel.IssueNum, slug)

	// Create worktree
	worktreePath, err := r.Worktree.Create(ctx, branchName)
	if err != nil {
		if updateErr := r.Queue.Update(vessel.ID, queue.StateFailed, fmt.Sprintf("create worktree: %v", err)); updateErr != nil {
			log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
		}
		return "failed"
	}

	// Build and launch claude command
	cmd, args, err := buildCommand(r.Config, &vessel)
	if err != nil {
		if updateErr := r.Queue.Update(vessel.ID, queue.StateFailed, fmt.Sprintf("build command: %v", err)); updateErr != nil {
			log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
		}
		return "failed"
	}

	runErr := r.Runner.RunProcess(ctx, worktreePath, cmd, args...)

	if runErr != nil {
		if updateErr := r.Queue.Update(vessel.ID, queue.StateFailed, runErr.Error()); updateErr != nil {
			log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
		}
		return "failed"
	}

	if updateErr := r.Queue.Update(vessel.ID, queue.StateCompleted, ""); updateErr != nil {
		log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
	}
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
func buildCommand(cfg *config.Config, vessel *queue.Vessel) (string, []string, error) {
	tmpl, err := template.New("cmd").Parse(cfg.Claude.Template)
	if err != nil {
		return "", nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cmdVars{
		Command:  cfg.Claude.Command,
		Skill:    vessel.Skill,
		IssueURL: vessel.IssueURL,
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
		clean = strings.TrimRight(clean, "-")
	}
	if clean == "" {
		clean = "task"
	}
	return clean
}
