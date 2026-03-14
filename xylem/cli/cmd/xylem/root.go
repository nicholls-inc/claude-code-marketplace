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
		Short:         "Autonomous issue agent scheduler",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			for _, tool := range []string{"gh", "git"} {
				if _, err := exec.LookPath(tool); err != nil {
					return fmt.Errorf("error: %s not found on PATH", tool)
				}
			}

			configPath := viper.GetString("config")
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("error loading config %s: %w", configPath, err)
			}

			queueFile := filepath.Join(cfg.StateDir, "queue.jsonl")
			deps = &appDeps{
				cfg: cfg,
				q:   queue.New(queueFile),
				wt:  worktree.New(".", &realCmdRunner{}),
			}
			return nil
		},
	}

	cmd.PersistentFlags().String("config", ".xylem.yml", "Config file path")
	viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config")) //nolint:errcheck
	viper.SetEnvPrefix("XYLEM")
	viper.AutomaticEnv()

	cmd.AddCommand(
		newScanCmd(),
		newDrainCmd(),
		newStatusCmd(),
		newPauseCmd(),
		newResumeCmd(),
		newCancelCmd(),
		newCleanupCmd(),
	)

	return cmd
}
