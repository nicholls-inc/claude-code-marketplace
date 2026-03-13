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

func validConfig() *Config {
	return &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		},
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		Claude: ClaudeConfig{
			Command:  "claude",
			Template: defaultClaudeTemplate,
		},
	}
}

func TestValidateMissingRepo(t *testing.T) {
	cfg := validConfig()
	cfg.Repo = ""

	err := cfg.Validate()
	requireErrorContains(t, err, "repo")
}

func TestValidateEmptyTasks(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = nil

	err := cfg.Validate()
	requireErrorContains(t, err, "tasks")
}

func TestValidateZeroConcurrency(t *testing.T) {
	cfg := validConfig()
	cfg.Concurrency = 0

	err := cfg.Validate()
	requireErrorContains(t, err, "concurrency")
}

func TestValidateInvalidTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = "invalid"

	err := cfg.Validate()
	requireErrorContains(t, err, "timeout")
}

func TestValidateTaskMissingLabels(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = map[string]Task{
		"fix-bugs": {Skill: "fix-bug"},
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "labels")
}

func TestValidateTaskMissingSkill(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = map[string]Task{
		"fix-bugs": {Labels: []string{"bug"}},
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

func TestValidateZeroMaxTurns(t *testing.T) {
	cfg := validConfig()
	cfg.MaxTurns = 0

	err := cfg.Validate()
	requireErrorContains(t, err, "max_turns")
}

func TestValidateNegativeMaxTurns(t *testing.T) {
	cfg := validConfig()
	cfg.MaxTurns = -5

	err := cfg.Validate()
	requireErrorContains(t, err, "max_turns")
}

func TestValidateInvalidTemplate(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.Template = "{{.Broken"

	err := cfg.Validate()
	requireErrorContains(t, err, "claude.template")
}

func TestValidateTimeoutTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = "5s"

	err := cfg.Validate()
	requireErrorContains(t, err, "timeout must be at least")
}

func TestValidateMalformedRepo(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want string
	}{
		{"no slash", "ownername", "owner/name format"},
		{"trailing slash", "owner/", "owner/name format"},
		{"leading slash", "/name", "owner/name format"},
		{"multiple slashes", "owner/name/extra", "owner/name format"},
		{"just slash", "/", "owner/name format"},
		{"whitespace parts", " / ", "owner/name format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Repo = tt.repo

			err := cfg.Validate()
			requireErrorContains(t, err, tt.want)
		})
	}
}

func TestLoadZeroMaxTurnsInYAML(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    skill: fix-bug
max_turns: 0
`)

	_, err := Load(path)
	requireErrorContains(t, err, "max_turns")
}

func TestLoadInvalidTemplateInYAML(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    skill: fix-bug
claude:
  template: "{{.Broken"
`)

	_, err := Load(path)
	requireErrorContains(t, err, "claude.template")
}

func TestLoadLowTimeoutInYAML(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    skill: fix-bug
timeout: "1s"
`)

	_, err := Load(path)
	requireErrorContains(t, err, "timeout must be at least")
}

func TestValidateNegativeConcurrency(t *testing.T) {
	cfg := validConfig()
	cfg.Concurrency = -1

	err := cfg.Validate()
	requireErrorContains(t, err, "concurrency")
}

func TestValidateTaskSkillWhitespaceOnly(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = map[string]Task{
		"fix-bugs": {Labels: []string{"bug"}, Skill: "  "},
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "skill")
}

func TestValidateTimeoutExactlyMinimum(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = "30s" // exactly the minimum

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected 30s timeout to be valid, got: %v", err)
	}
}

func TestValidateTimeoutJustBelowMinimum(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = "29s"

	err := cfg.Validate()
	requireErrorContains(t, err, "timeout must be at least")
}

func TestLoadYAMLWithUnknownFields(t *testing.T) {
	// Go's yaml.v3 ignores unknown fields by default.
	// Verify this doesn't cause a failure.
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    skill: fix-bug
unknown_field: some_value
nested_unknown:
  key: value
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected unknown fields to be ignored, got: %v", err)
	}
	if cfg.Repo != "owner/name" {
		t.Fatalf("Repo = %q, want owner/name", cfg.Repo)
	}
}

func TestValidateMultipleTasks(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = map[string]Task{
		"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		"features": {Labels: []string{"enhancement"}, Skill: "implement-feature"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config with multiple tasks, got: %v", err)
	}
}

func TestValidateMultipleTasksOneInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = map[string]Task{
		"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		"broken":   {Labels: []string{}, Skill: "fix-bug"}, // no labels
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "labels")
}
