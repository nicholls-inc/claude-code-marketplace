package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfigFile(t *testing.T, yaml string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func requireErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

func TestLoadValid(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug, ready-for-work]
    skill: fix-bug
concurrency: 3
max_turns: 30
timeout: "15m"
state_dir: ".pit-crew"
exclude: [wontfix, duplicate]
claude:
  command: "claude"
  template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Repo != "owner/name" {
		t.Fatalf("Repo = %q, want owner/name", cfg.Repo)
	}

	task, ok := cfg.Tasks["fix-bugs"]
	if !ok {
		t.Fatalf("missing task fix-bugs")
	}

	if len(task.Labels) != 2 || task.Labels[0] != "bug" || task.Labels[1] != "ready-for-work" {
		t.Fatalf("task labels = %#v, want [bug ready-for-work]", task.Labels)
	}

	if task.Skill != "fix-bug" {
		t.Fatalf("task skill = %q, want fix-bug", task.Skill)
	}

	if cfg.Concurrency != 3 {
		t.Fatalf("Concurrency = %d, want 3", cfg.Concurrency)
	}

	if cfg.MaxTurns != 30 {
		t.Fatalf("MaxTurns = %d, want 30", cfg.MaxTurns)
	}

	if cfg.Timeout != "15m" {
		t.Fatalf("Timeout = %q, want 15m", cfg.Timeout)
	}

	if cfg.StateDir != ".pit-crew" {
		t.Fatalf("StateDir = %q, want .pit-crew", cfg.StateDir)
	}

	if len(cfg.Exclude) != 2 || cfg.Exclude[0] != "wontfix" || cfg.Exclude[1] != "duplicate" {
		t.Fatalf("Exclude = %#v, want [wontfix duplicate]", cfg.Exclude)
	}

	if cfg.Claude.Command != "claude" {
		t.Fatalf("Claude.Command = %q, want claude", cfg.Claude.Command)
	}

	wantTemplate := "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"
	if cfg.Claude.Template != wantTemplate {
		t.Fatalf("Claude.Template = %q, want %q", cfg.Claude.Template, wantTemplate)
	}
}

func TestLoadDefaults(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    skill: fix-bug
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Concurrency != 2 {
		t.Fatalf("Concurrency = %d, want 2", cfg.Concurrency)
	}

	if cfg.MaxTurns != 50 {
		t.Fatalf("MaxTurns = %d, want 50", cfg.MaxTurns)
	}

	if cfg.Timeout != "30m" {
		t.Fatalf("Timeout = %q, want 30m", cfg.Timeout)
	}

	if cfg.StateDir != ".pit-crew" {
		t.Fatalf("StateDir = %q, want .pit-crew", cfg.StateDir)
	}

	wantExclude := []string{"wontfix", "duplicate", "in-progress", "no-bot"}
	if len(cfg.Exclude) != len(wantExclude) {
		t.Fatalf("Exclude length = %d, want %d (%#v)", len(cfg.Exclude), len(wantExclude), cfg.Exclude)
	}

	for i := range wantExclude {
		if cfg.Exclude[i] != wantExclude[i] {
			t.Fatalf("Exclude[%d] = %q, want %q", i, cfg.Exclude[i], wantExclude[i])
		}
	}

	if cfg.Claude.Command != "claude" {
		t.Fatalf("Claude.Command = %q, want claude", cfg.Claude.Command)
	}

	wantTemplate := "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"
	if cfg.Claude.Template != wantTemplate {
		t.Fatalf("Claude.Template = %q, want %q", cfg.Claude.Template, wantTemplate)
	}
}

func TestValidateMissingRepo(t *testing.T) {
	cfg := &Config{
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		},
		Concurrency: 2,
		Timeout:     "30m",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "repo")
}

func TestValidateEmptyTasks(t *testing.T) {
	cfg := &Config{
		Repo:        "owner/name",
		Concurrency: 2,
		Timeout:     "30m",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "tasks")
}

func TestValidateZeroConcurrency(t *testing.T) {
	cfg := &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		},
		Concurrency: 0,
		Timeout:     "30m",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "concurrency")
}

func TestValidateInvalidTimeout(t *testing.T) {
	cfg := &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		},
		Concurrency: 2,
		Timeout:     "invalid",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "timeout")
}

func TestValidateTaskMissingLabels(t *testing.T) {
	cfg := &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Skill: "fix-bug"},
		},
		Concurrency: 2,
		Timeout:     "30m",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "labels")
}

func TestValidateTaskMissingSkill(t *testing.T) {
	cfg := &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}},
		},
		Concurrency: 2,
		Timeout:     "30m",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "skill")
}

func TestMalformedYAML(t *testing.T) {
	path := writeConfigFile(t, "repo: [owner/name\n")

	_, err := Load(path)
	if err == nil {
		t.Fatalf("expected error for malformed yaml")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
