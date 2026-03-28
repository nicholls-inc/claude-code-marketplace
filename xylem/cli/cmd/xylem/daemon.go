package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/runner"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/scanner"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run continuous scan-drain loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdDaemon(deps.cfg, deps.q, deps.wt)
		},
	}
	return cmd
}

func cmdDaemon(cfg *config.Config, q *queue.Queue, wt *worktree.Manager) error {
	scanInterval, drainInterval := parseDaemonIntervals(cfg.Daemon)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return daemonLoop(ctx, cfg, q, wt, scanInterval, drainInterval)
}

func parseDaemonIntervals(dc config.DaemonConfig) (scan, drain time.Duration) {
	scan = 60 * time.Second
	drain = 30 * time.Second

	if dc.ScanInterval != "" {
		if d, err := time.ParseDuration(dc.ScanInterval); err == nil {
			scan = d
		}
	}
	if dc.DrainInterval != "" {
		if d, err := time.ParseDuration(dc.DrainInterval); err == nil {
			drain = d
		}
	}
	return scan, drain
}

// daemonLoop is the core loop extracted for testability. It accepts an
// externally-controlled context so tests can cancel it without signals.
func daemonLoop(ctx context.Context, cfg *config.Config, q *queue.Queue, wt *worktree.Manager, scanInterval, drainInterval time.Duration) error {
	tickInterval := scanInterval
	if drainInterval < tickInterval {
		tickInterval = drainInterval
	}

	var lastScan, lastDrain time.Time

	log.Printf("daemon started: scan_interval=%s drain_interval=%s", scanInterval, drainInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("daemon: received shutdown signal")
			return nil
		default:
		}

		now := time.Now()

		if now.Sub(lastScan) >= scanInterval {
			scanResult, err := runScan(ctx, cfg, q)
			if err != nil {
				log.Printf("daemon: scan error: %v", err)
			} else {
				lastScan = now
				log.Printf("daemon: scan complete — added=%d skipped=%d", scanResult.Added, scanResult.Skipped)
			}
		}

		if now.Sub(lastDrain) >= drainInterval {
			drainResult, err := runDrain(ctx, cfg, q, wt)
			if err != nil {
				log.Printf("daemon: drain error: %v", err)
			} else {
				lastDrain = now
				log.Printf("daemon: drain complete — completed=%d failed=%d skipped=%d",
					drainResult.Completed, drainResult.Failed, drainResult.Skipped)
			}
		}

		logTickSummary(q)

		select {
		case <-ctx.Done():
			log.Println("daemon: shutting down gracefully")
			return nil
		case <-time.After(tickInterval):
		}
	}
}

func runScan(ctx context.Context, cfg *config.Config, q *queue.Queue) (scanner.ScanResult, error) {
	cmdRunner := &realCmdRunner{}
	s := scanner.New(cfg, q, cmdRunner)
	return s.Scan(ctx)
}

func runDrain(ctx context.Context, cfg *config.Config, q *queue.Queue, wt *worktree.Manager) (runner.DrainResult, error) {
	cmdRunner := &realCmdRunner{}
	r := runner.New(cfg, q, wt, cmdRunner)
	r.Sources = buildSourceMap(cfg, q, cmdRunner)
	return r.Drain(ctx)
}

func logTickSummary(q *queue.Queue) {
	vessels, err := q.List()
	if err != nil {
		return
	}
	counts := make(map[queue.VesselState]int)
	for _, v := range vessels {
		counts[v.State]++
	}
	log.Printf("daemon: tick summary — pending=%d running=%d completed=%d failed=%d",
		counts[queue.StatePending], counts[queue.StateRunning],
		counts[queue.StateCompleted], counts[queue.StateFailed])
}
