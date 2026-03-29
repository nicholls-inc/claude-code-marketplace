package main

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

type appDeps struct {
	cfg *config.Config
	q   *queue.Queue
	wt  *worktree.Manager
}

var deps *appDeps

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "xylem",
		Short:         "Autonomous Claude Code session scheduler",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" {
				return nil
			}

			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("error: git not found on PATH")
			}

			configPath := viper.GetString("config")
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("error loading config %s: %w", configPath, err)
			}

			// Only require gh if a GitHub source is configured
			if hasGitHubSource(cfg) {
				if _, err := exec.LookPath("gh"); err != nil {
					return fmt.Errorf("error: gh not found on PATH (required for github source)")
				}
			}

			queueFile := filepath.Join(cfg.StateDir, "queue.jsonl")
			wt := worktree.New(".", &realCmdRunner{})
			wt.DefaultBranch = cfg.DefaultBranch
			deps = &appDeps{
				cfg: cfg,
				q:   queue.New(queueFile),
				wt:  wt,
			}
			return nil
		},
	}

	cmd.PersistentFlags().String("config", ".xylem.yml", "Config file path")
	viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config")) //nolint:errcheck
	viper.SetEnvPrefix("XYLEM")
	viper.AutomaticEnv()

	cmd.AddCommand(
		newInitCmd(),
		newScanCmd(),
		newDrainCmd(),
		newEnqueueCmd(),
		newStatusCmd(),
		newPauseCmd(),
		newResumeCmd(),
		newCancelCmd(),
		newCleanupCmd(),
		newDaemonCmd(),
		newRetryCmd(),
	)

	return cmd
}

func hasGitHubSource(cfg *config.Config) bool {
	for _, src := range cfg.Sources {
		if src.Type == "github" {
			return true
		}
	}
	return false
}
