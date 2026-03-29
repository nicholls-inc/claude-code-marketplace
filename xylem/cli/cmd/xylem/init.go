package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultStateDir = ".xylem"

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap .xylem.yml config and .xylem/ state directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			configPath := viper.GetString("config")
			return cmdInit(configPath, force)
		},
	}
	cmd.Flags().Bool("force", false, "Overwrite existing .xylem.yml")
	return cmd
}

func cmdInit(configPath string, force bool) error {
	// Write scaffold config
	wrote, err := writeScaffoldConfig(configPath, force)
	if err != nil {
		return err
	}
	if wrote {
		fmt.Printf("Created %s\n", configPath)
	} else {
		fmt.Printf("%s already exists (use --force to overwrite)\n", configPath)
	}

	// Create state directory
	if err := os.MkdirAll(defaultStateDir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	fmt.Printf("Ensured %s/ directory exists\n", defaultStateDir)

	// Write .gitignore unconditionally
	gitignorePath := filepath.Join(defaultStateDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("*\n!.gitignore\n"), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s with your repo and task config\n", configPath)
	fmt.Println("  2. Run `xylem scan --dry-run` to preview what would be queued")
	fmt.Println("  3. Run `xylem scan && xylem drain` to start processing")
	return nil
}

func writeScaffoldConfig(configPath string, force bool) (bool, error) {
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return false, nil
		}
	}

	repo := detectGitHubRepo()
	if repo == "" {
		repo = "owner/name"
	}

	content := fmt.Sprintf(`# xylem configuration
# Docs: https://github.com/nicholls-inc/claude-code-marketplace/tree/main/xylem

sources:
  bugs:
    type: github
    repo: %s
    exclude: [wontfix, duplicate, in-progress, no-bot]
    tasks:
      fix-bugs:
        labels: [bug, ready-for-work]
        skill: fix-bug
  # features:
  #   type: github
  #   repo: %s
  #   exclude: [wontfix, duplicate, in-progress, no-bot]
  #   tasks:
  #     implement-features:
  #       labels: [enhancement, low-effort, ready-for-work]
  #       skill: implement-feature

concurrency: 2
max_turns: 50
timeout: "30m"
state_dir: ".xylem"

claude:
  command: "claude"
  flags: "--bare --dangerously-skip-permissions"
  env:
    ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
`, repo, repo)

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}
	return true, nil
}

// parseGitHubRepo extracts "owner/name" from a GitHub remote URL.
// Returns "" for non-GitHub or malformed URLs.
func parseGitHubRepo(remoteURL string) string {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return ""
	}

	// SSH: git@github.com:owner/name.git
	sshRe := regexp.MustCompile(`^git@github\.com:([^/]+/[^/]+?)(?:\.git)?$`)
	if m := sshRe.FindStringSubmatch(remoteURL); m != nil {
		return m[1]
	}

	// HTTPS: https://github.com/owner/name.git
	httpsRe := regexp.MustCompile(`^https://github\.com/([^/]+/[^/]+?)(?:\.git)?$`)
	if m := httpsRe.FindStringSubmatch(remoteURL); m != nil {
		return m[1]
	}

	// ssh://git@github.com/owner/name.git
	sshProtoRe := regexp.MustCompile(`^ssh://git@github\.com/([^/]+/[^/]+?)(?:\.git)?$`)
	if m := sshProtoRe.FindStringSubmatch(remoteURL); m != nil {
		return m[1]
	}

	return ""
}

func detectGitHubRepo() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return parseGitHubRepo(string(out))
}
