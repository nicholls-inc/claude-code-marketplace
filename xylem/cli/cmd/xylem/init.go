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

	// Create HARNESS.md
	writeFileIfNeeded(filepath.Join(defaultStateDir, "HARNESS.md"), harnessContent, force)

	// Create workflow definitions
	writeFileIfNeeded(filepath.Join(defaultStateDir, "workflows", "fix-bug.yaml"), fixBugWorkflowContent, force)
	writeFileIfNeeded(filepath.Join(defaultStateDir, "workflows", "implement-feature.yaml"), implementFeatureWorkflowContent, force)

	// Create prompt templates
	for _, wf := range []string{"fix-bug", "implement-feature"} {
		for _, phase := range []string{"analyze", "plan", "implement", "pr"} {
			writeFileIfNeeded(filepath.Join(defaultStateDir, "prompts", wf, phase+".md"), promptContent(wf, phase), force)
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s with your repo and task config\n", configPath)
	fmt.Printf("  2. Edit %s/HARNESS.md with your project details\n", defaultStateDir)
	fmt.Println("  3. Run `xylem scan --dry-run` to preview what would be queued")
	fmt.Println("  4. Run `xylem scan && xylem drain` to start processing")
	return nil
}

func writeFileIfNeeded(path string, content string, force bool) {
	if !force {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("skipped: %s (already exists)\n", path)
			return
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("warning: failed to create directory for %s: %v\n", path, err)
		return
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Printf("warning: failed to write %s: %v\n", path, err)
		return
	}
	fmt.Printf("Created %s\n", path)
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
        workflow: fix-bug
  # features:
  #   type: github
  #   repo: %s
  #   exclude: [wontfix, duplicate, in-progress, no-bot]
  #   tasks:
  #     implement-features:
  #       labels: [enhancement, low-effort, ready-for-work]
  #       workflow: implement-feature

concurrency: 2
max_turns: 50
timeout: "30m"
state_dir: ".xylem"

claude:
  command: "claude"
  flags: "--bare --dangerously-skip-permissions"
  env:
    ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
  # allowed_tools:
  #   - "Bash(gh issue view *)"
  #   - "Bash(gh pr create *)"
  #   - "WebFetch"
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

const harnessContent = `# Project Overview
<!-- Describe what this project does -->

# Architecture
<!-- Describe the codebase structure -->

# Build & Test
<!-- List the exact build, test, and lint commands -->

# Golden Principles
<!-- List rules the agent must always follow — e.g., "always run gofmt", "never modify generated files" -->

# Dependencies
<!-- Note any external services or tools the agent needs -->
`

const fixBugWorkflowContent = `name: fix-bug
description: "Diagnose and fix a bug from a GitHub issue"
phases:
  - name: analyze
    prompt_file: .xylem/prompts/fix-bug/analyze.md
    max_turns: 5
  - name: plan
    prompt_file: .xylem/prompts/fix-bug/plan.md
    max_turns: 3
  - name: implement
    prompt_file: .xylem/prompts/fix-bug/implement.md
    max_turns: 15
    gate:
      type: command
      run: "make test"
      retries: 2
  - name: pr
    prompt_file: .xylem/prompts/fix-bug/pr.md
    max_turns: 3
`

const implementFeatureWorkflowContent = `name: implement-feature
description: "Implement a feature from a GitHub issue"
phases:
  - name: analyze
    prompt_file: .xylem/prompts/implement-feature/analyze.md
    max_turns: 5
  - name: plan
    prompt_file: .xylem/prompts/implement-feature/plan.md
    max_turns: 3
    gate:
      type: label
      wait_for: "plan-approved"
      timeout: "24h"
  - name: implement
    prompt_file: .xylem/prompts/implement-feature/implement.md
    max_turns: 15
    gate:
      type: command
      run: "make test"
      retries: 2
  - name: pr
    prompt_file: .xylem/prompts/implement-feature/pr.md
    max_turns: 3
`

const analyzePrompt = `Analyze the following GitHub issue and identify the relevant code.

Issue: {{.Issue.Title}}
URL: {{.Issue.URL}}

{{.Issue.Body}}

Read the codebase and identify:
1. Which files are relevant to this issue
2. The root cause (for bugs) or the requirements (for features)
3. Any dependencies or constraints

Write your analysis clearly and concisely.
`

const planPrompt = `Based on the analysis from the previous phase, create an implementation plan.

Issue: {{.Issue.Title}}
URL: {{.Issue.URL}}

## Previous Analysis
{{.PreviousOutputs.analyze}}

Write a step-by-step plan that includes:
1. What changes need to be made and in which files
2. The order of changes
3. Any tests that need to be added or updated
4. Potential risks or edge cases
`

const implementPrompt = `Implement the changes according to the plan.

Issue: {{.Issue.Title}}
URL: {{.Issue.URL}}

## Analysis
{{.PreviousOutputs.analyze}}

## Plan
{{.PreviousOutputs.plan}}

{{if .GateResult}}
## Previous Gate Failure
The following gate check failed after the previous attempt. Fix the issues and try again:

{{.GateResult}}
{{end}}

Implement the changes now. Follow the plan precisely.
`

const prPrompt = `Create a pull request for the changes.

Issue: {{.Issue.Title}}
URL: {{.Issue.URL}}

## Analysis
{{.PreviousOutputs.analyze}}

## Plan
{{.PreviousOutputs.plan}}

Commit all changes with a clear commit message, push the branch, and create a PR using:
gh pr create --title "<descriptive title>" --body "<summary of changes, linking to {{.Issue.URL}}>"
`

func promptContent(wf, phase string) string {
	_ = wf // All workflows share the same prompt templates per phase.
	switch phase {
	case "analyze":
		return analyzePrompt
	case "plan":
		return planPrompt
	case "implement":
		return implementPrompt
	case "pr":
		return prPrompt
	default:
		return fmt.Sprintf("# %s\n<!-- Add your %s prompt here -->\n", phase, phase)
	}
}
