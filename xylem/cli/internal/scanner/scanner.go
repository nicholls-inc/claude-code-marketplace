package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
)

// CommandRunner abstracts shell calls for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Scanner scans configured sources for actionable tasks and enqueues vessels.
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

// New creates a Scanner.
func New(cfg *config.Config, q *queue.Queue, runner CommandRunner) *Scanner {
	return &Scanner{Config: cfg, Queue: q, CmdRunner: runner}
}

// Scan queries configured sources, filters candidates, and enqueues new vessels.
func (s *Scanner) Scan(ctx context.Context) (ScanResult, error) {
	pauseMarker := filepath.Join(s.Config.StateDir, "paused")
	if _, err := os.Stat(pauseMarker); err == nil {
		return ScanResult{Paused: true}, nil
	}

	var result ScanResult
	sources := s.buildSources()

	for _, src := range sources {
		vessels, err := src.Scan(ctx)
		if err != nil {
			return result, err
		}
		for _, vessel := range vessels {
			// Dedup: skip if already enqueued (from another source or task)
			if s.Queue.HasRef(vessel.Ref) {
				result.Skipped++
				continue
			}
			if err := s.Queue.Enqueue(vessel); err != nil {
				return result, err
			}
			result.Added++
		}
	}
	return result, nil
}

// buildSources constructs Source implementations from the config.
func (s *Scanner) buildSources() []source.Source {
	var sources []source.Source
	for _, srcCfg := range s.Config.Sources {
		if srcCfg.Type == "github" {
			tasks := make(map[string]source.GitHubTask, len(srcCfg.Tasks))
			for name, t := range srcCfg.Tasks {
				tasks[name] = source.GitHubTask{
					Labels: t.Labels,
					Skill:  t.Skill,
				}
			}
			sources = append(sources, &source.GitHub{
				Repo:      srcCfg.Repo,
				Tasks:     tasks,
				Exclude:   srcCfg.Exclude,
				Queue:     s.Queue,
				CmdRunner: s.CmdRunner,
			})
		}
	}
	return sources
}
