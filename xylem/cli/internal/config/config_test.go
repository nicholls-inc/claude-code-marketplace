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
    workflow: fix-bug
concurrency: 3
max_turns: 30
timeout: "15m"
state_dir: ".xylem"
exclude: [wontfix, duplicate]
claude:
  command: "claude"
  flags: "--bare"
  env:
    ANTHROPIC_API_KEY: "test-key"
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

	if task.Workflow != "fix-bug" {
		t.Fatalf("task workflow = %q, want fix-bug", task.Workflow)
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

	if cfg.StateDir != ".xylem" {
		t.Fatalf("StateDir = %q, want .xylem", cfg.StateDir)
	}

	if len(cfg.Exclude) != 2 || cfg.Exclude[0] != "wontfix" || cfg.Exclude[1] != "duplicate" {
		t.Fatalf("Exclude = %#v, want [wontfix duplicate]", cfg.Exclude)
	}

	if cfg.Claude.Command != "claude" {
		t.Fatalf("Claude.Command = %q, want claude", cfg.Claude.Command)
	}

	if cfg.Claude.Flags != "--bare" {
		t.Fatalf("Claude.Flags = %q, want --bare", cfg.Claude.Flags)
	}

	if cfg.Claude.Env["ANTHROPIC_API_KEY"] != "test-key" {
		t.Fatalf("Claude.Env[ANTHROPIC_API_KEY] = %q, want test-key", cfg.Claude.Env["ANTHROPIC_API_KEY"])
	}

	// Legacy config should be normalized into Sources
	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source after normalization, got %d", len(cfg.Sources))
	}
	ghSrc, ok := cfg.Sources["github"]
	if !ok {
		t.Fatal("expected github source after normalization")
	}
	if ghSrc.Repo != "owner/name" {
		t.Fatalf("source repo = %q, want owner/name", ghSrc.Repo)
	}
}

func TestLoadDefaults(t *testing.T) {
	path := writeConfigFile(t, `repo: owner/name
tasks:
  fix-bugs:
    labels: [bug]
    workflow: fix-bug
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

	if cfg.StateDir != ".xylem" {
		t.Fatalf("StateDir = %q, want .xylem", cfg.StateDir)
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

	if cfg.Claude.Flags != "" {
		t.Fatalf("Claude.Flags = %q, want empty", cfg.Claude.Flags)
	}

	// Daemon defaults
	if cfg.Daemon.ScanInterval != "60s" {
		t.Fatalf("Daemon.ScanInterval = %q, want 60s", cfg.Daemon.ScanInterval)
	}
	if cfg.Daemon.DrainInterval != "30s" {
		t.Fatalf("Daemon.DrainInterval = %q, want 30s", cfg.Daemon.DrainInterval)
	}

	// Legacy config should be normalized into Sources
	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source after normalization, got %d", len(cfg.Sources))
	}
}

func validConfig() *Config {
	return &Config{
		Repo: "owner/name",
		Tasks: map[string]Task{
			"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"},
		},
		Sources: map[string]SourceConfig{
			"github": {
				Type: "github",
				Repo: "owner/name",
				Tasks: map[string]Task{
					"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"},
				},
			},
		},
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		Claude: ClaudeConfig{
			Command: "claude",
		},
	}
}

func TestValidateMissingRepoInGitHubSource(t *testing.T) {
	cfg := &Config{
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		Claude:      ClaudeConfig{Command: "claude"},
		Sources: map[string]SourceConfig{
			"github": {Type: "github", Repo: "", Tasks: map[string]Task{
				"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"},
			}},
		},
	}
	err := cfg.Validate()
	requireErrorContains(t, err, "repo")
}

func TestValidateNoSourcesNoRepoIsValid(t *testing.T) {
	cfg := &Config{
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		Claude:      ClaudeConfig{Command: "claude"},
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid config with no sources (enqueue-only), got: %v", err)
	}
}

func TestValidateEmptyTasksInGitHubSource(t *testing.T) {
	cfg := &Config{
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		Claude:      ClaudeConfig{Command: "claude"},
		Sources: map[string]SourceConfig{
			"github": {Type: "github", Repo: "owner/name", Tasks: nil},
		},
	}
	err := cfg.Validate()
	requireErrorContains(t, err, "task")
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
	cfg.Sources["github"] = SourceConfig{
		Type: "github", Repo: "owner/name",
		Tasks: map[string]Task{"fix-bugs": {Workflow: "fix-bug"}},
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "labels")
}

func TestValidateTaskMissingWorkflow(t *testing.T) {
	cfg := validConfig()
	cfg.Sources["github"] = SourceConfig{
		Type: "github", Repo: "owner/name",
		Tasks: map[string]Task{"fix-bugs": {Labels: []string{"bug"}}},
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "workflow")
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

func TestValidateTemplateRejected(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.Template = "{{.Command}} -p \"/{{.Workflow}} {{.Ref}}\" --max-turns {{.MaxTurns}}"

	err := cfg.Validate()
	requireErrorContains(t, err, "claude.template is no longer supported")
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
			cfg := &Config{
				Concurrency: 2,
				MaxTurns:    50,
				Timeout:     "30m",
				Claude:      ClaudeConfig{Command: "claude"},
				Sources: map[string]SourceConfig{
					"github": {
						Type: "github", Repo: tt.repo,
						Tasks: map[string]Task{"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"}},
					},
				},
			}
			err := cfg.Validate()
			requireErrorContains(t, err, tt.want)
		})
	}
}

func TestValidateTaskWorkflowWhitespaceOnly(t *testing.T) {
	cfg := validConfig()
	cfg.Sources["github"] = SourceConfig{
		Type: "github", Repo: "owner/name",
		Tasks: map[string]Task{"fix-bugs": {Labels: []string{"bug"}, Workflow: "  "}},
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "workflow")
}

func TestValidateTimeoutExactlyMinimum(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = "30s" // exactly the minimum

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected 30s timeout to be valid, got: %v", err)
	}
}

func TestValidateMultipleTasksOneInvalid(t *testing.T) {
	cfg := validConfig()
	tasks := map[string]Task{
		"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"},
		"broken":   {Labels: []string{}, Workflow: "fix-bug"}, // no labels
	}
	cfg.Sources["github"] = SourceConfig{Type: "github", Repo: "owner/name", Tasks: tasks}

	err := cfg.Validate()
	requireErrorContains(t, err, "labels")
}

func TestValidateAllowedToolsRejected(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.AllowedTools = []string{"WebFetch"}

	err := cfg.Validate()
	requireErrorContains(t, err, "claude.allowed_tools is no longer supported")
}

func TestValidateAllowedToolsEmpty(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.AllowedTools = []string{}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected empty allowed_tools to be valid, got: %v", err)
	}
}

func TestValidateBareWithoutAPIKey(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.Flags = "--bare --dangerously-skip-permissions"

	err := cfg.Validate()
	requireErrorContains(t, err, "--bare requires ANTHROPIC_API_KEY in claude.env")
}

func TestValidateBareWithAPIKey(t *testing.T) {
	cfg := validConfig()
	cfg.Claude.Flags = "--bare --dangerously-skip-permissions"
	cfg.Claude.Env = map[string]string{"ANTHROPIC_API_KEY": "test-key"}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid config with --bare and API key, got: %v", err)
	}
}

func TestValidateDaemonIntervalValid(t *testing.T) {
	cfg := validConfig()
	cfg.Daemon = DaemonConfig{
		ScanInterval:  "60s",
		DrainInterval: "30s",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid daemon config, got: %v", err)
	}
}

func TestValidateDaemonScanIntervalInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Daemon = DaemonConfig{
		ScanInterval: "not-a-duration",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "daemon.scan_interval must be a valid duration")
}

func TestValidateDaemonDrainIntervalInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Daemon = DaemonConfig{
		DrainInterval: "bad",
	}

	err := cfg.Validate()
	requireErrorContains(t, err, "daemon.drain_interval must be a valid duration")
}

func TestLoadWithFlagsAndEnv(t *testing.T) {
	path := writeConfigFile(t, `sources:
  github:
    type: github
    repo: owner/name
    tasks:
      fix-bugs:
        labels: [bug]
        workflow: fix-bug
concurrency: 2
max_turns: 50
timeout: "30m"
claude:
  command: "claude"
  flags: "--bare --dangerously-skip-permissions"
  env:
    ANTHROPIC_API_KEY: "sk-test-123"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Claude.Flags != "--bare --dangerously-skip-permissions" {
		t.Fatalf("Claude.Flags = %q, want --bare --dangerously-skip-permissions", cfg.Claude.Flags)
	}
	if cfg.Claude.Env["ANTHROPIC_API_KEY"] != "sk-test-123" {
		t.Fatalf("Claude.Env[ANTHROPIC_API_KEY] = %q, want sk-test-123", cfg.Claude.Env["ANTHROPIC_API_KEY"])
	}
}

func TestLoadRejectsOldTemplate(t *testing.T) {
	path := writeConfigFile(t, `sources:
  github:
    type: github
    repo: owner/name
    tasks:
      fix-bugs:
        labels: [bug]
        workflow: fix-bug
concurrency: 2
max_turns: 50
timeout: "30m"
claude:
  command: "claude"
  template: "{{.Command}} -p \"/{{.Workflow}} {{.Ref}}\" --max-turns {{.MaxTurns}}"
`)

	_, err := Load(path)
	requireErrorContains(t, err, "claude.template is no longer supported")
}

func TestLoadRejectsOldAllowedTools(t *testing.T) {
	path := writeConfigFile(t, `sources:
  github:
    type: github
    repo: owner/name
    tasks:
      fix-bugs:
        labels: [bug]
        workflow: fix-bug
concurrency: 2
max_turns: 50
timeout: "30m"
claude:
  command: "claude"
  allowed_tools:
    - "Bash(gh issue view *)"
`)

	_, err := Load(path)
	requireErrorContains(t, err, "claude.allowed_tools is no longer supported")
}

func TestLoadDaemonConfig(t *testing.T) {
	path := writeConfigFile(t, `sources:
  github:
    type: github
    repo: owner/name
    tasks:
      fix-bugs:
        labels: [bug]
        workflow: fix-bug
concurrency: 2
max_turns: 50
timeout: "30m"
claude:
  command: "claude"
daemon:
  scan_interval: "120s"
  drain_interval: "45s"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Daemon.ScanInterval != "120s" {
		t.Fatalf("Daemon.ScanInterval = %q, want 120s", cfg.Daemon.ScanInterval)
	}
	if cfg.Daemon.DrainInterval != "45s" {
		t.Fatalf("Daemon.DrainInterval = %q, want 45s", cfg.Daemon.DrainInterval)
	}
}
