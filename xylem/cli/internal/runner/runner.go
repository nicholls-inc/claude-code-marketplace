package runner

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
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
	Sources  map[string]source.Source
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
	// Look up source for this vessel
	src := r.resolveSource(vessel.Source)

	// Source-specific start hook (e.g., add in-progress label)
	if err := src.OnStart(ctx, vessel); err != nil {
		log.Printf("warn: source OnStart for %s: %v", vessel.ID, err)
	}

	// Source-specific branch naming
	branchName := src.BranchName(vessel)

	// Create worktree
	worktreePath, err := r.Worktree.Create(ctx, branchName)
	if err != nil {
		if updateErr := r.Queue.Update(vessel.ID, queue.StateFailed, err.Error()); updateErr != nil {
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

func (r *Runner) resolveSource(name string) source.Source {
	if r.Sources != nil {
		if src, ok := r.Sources[name]; ok {
			return src
		}
	}
	return &source.Manual{}
}

// cmdVars holds template substitution values.
type cmdVars struct {
	Command  string
	Skill    string
	Ref      string
	Prompt   string
	MaxTurns int
	Meta     map[string]string
	IssueURL string // backward-compat alias
}

// buildCommand renders the config template and returns (cmd, args).
func buildCommand(cfg *config.Config, vessel *queue.Vessel) (string, []string, error) {
	// Direct prompt mode: bypass template
	if vessel.Prompt != "" {
		prompt := vessel.Prompt
		if vessel.Ref != "" {
			prompt = fmt.Sprintf("Ref: %s\n\n%s", vessel.Ref, vessel.Prompt)
		}
		args := []string{"-p", prompt, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns)}
		return cfg.Claude.Command, args, nil
	}

	tmpl, err := template.New("cmd").Parse(cfg.Claude.Template)
	if err != nil {
		return "", nil, fmt.Errorf("parse template: %w", err)
	}

	vars := cmdVars{
		Command:  cfg.Claude.Command,
		Skill:    vessel.Skill,
		Ref:      vessel.Ref,
		MaxTurns: cfg.MaxTurns,
		Meta:     vessel.Meta,
	}
	// Backward compat: populate IssueURL for legacy templates
	if vessel.Source == "github-issue" {
		vars.IssueURL = vessel.Ref
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", nil, fmt.Errorf("execute template: %w", err)
	}
	parts := strings.Fields(buf.String())
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command from template")
	}
	return parts[0], parts[1:], nil
}
