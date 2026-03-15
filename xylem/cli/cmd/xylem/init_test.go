package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
)

func TestInitCreatesConfigAndStateDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".xylem.yml")
	stateDir := filepath.Join(dir, ".xylem")

	// Temporarily change to temp dir so defaultStateDir resolves there
	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	out := captureStdout(func() {
		err := cmdInit(configPath, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Config file created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	// State directory created
	if _, err := os.Stat(stateDir); err != nil {
		t.Errorf("state dir not created: %v", err)
	}

	// .gitignore created
	gitignore := filepath.Join(stateDir, ".gitignore")
	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("gitignore not created: %v", err)
	}
	if string(data) != "*\n!.gitignore\n" {
		t.Errorf("unexpected gitignore content: %q", string(data))
	}

	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' in output, got: %s", out)
	}
	if !strings.Contains(out, "Next steps") {
		t.Errorf("expected 'Next steps' in output, got: %s", out)
	}
}

func TestInitIdempotentWithoutForce(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".xylem.yml")

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	// Write existing config
	existing := "existing: true\n"
	os.WriteFile(configPath, []byte(existing), 0o644) //nolint:errcheck

	out := captureStdout(func() {
		err := cmdInit(configPath, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Config preserved
	data, _ := os.ReadFile(configPath)
	if string(data) != existing {
		t.Errorf("config was overwritten, got: %s", string(data))
	}

	if !strings.Contains(out, "already exists") {
		t.Errorf("expected 'already exists' message, got: %s", out)
	}

	// State dir still created
	stateDir := filepath.Join(dir, ".xylem")
	if _, err := os.Stat(stateDir); err != nil {
		t.Errorf("state dir not created: %v", err)
	}
}

func TestInitForceOverwritesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".xylem.yml")

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	// Write existing config
	os.WriteFile(configPath, []byte("old: true\n"), 0o644) //nolint:errcheck

	out := captureStdout(func() {
		err := cmdInit(configPath, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	data, _ := os.ReadFile(configPath)
	if strings.Contains(string(data), "old: true") {
		t.Errorf("config was not overwritten")
	}
	if !strings.Contains(string(data), "sources:") {
		t.Errorf("expected scaffold config, got: %s", string(data))
	}

	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' in output, got: %s", out)
	}
}

func TestInitStateDirAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".xylem.yml")
	stateDir := filepath.Join(dir, ".xylem")

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	// Pre-create state dir with a file
	os.MkdirAll(stateDir, 0o755) //nolint:errcheck
	os.WriteFile(filepath.Join(stateDir, "queue.jsonl"), []byte("existing"), 0o644) //nolint:errcheck

	err := cmdInit(configPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Existing file preserved
	data, _ := os.ReadFile(filepath.Join(stateDir, "queue.jsonl"))
	if string(data) != "existing" {
		t.Errorf("existing file in state dir was modified")
	}
}

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"SSH", "git@github.com:owner/repo.git", "owner/repo"},
		{"SSH no .git", "git@github.com:owner/repo", "owner/repo"},
		{"HTTPS", "https://github.com/owner/repo.git", "owner/repo"},
		{"HTTPS no .git", "https://github.com/owner/repo", "owner/repo"},
		{"ssh protocol", "ssh://git@github.com/owner/repo.git", "owner/repo"},
		{"ssh protocol no .git", "ssh://git@github.com/owner/repo", "owner/repo"},
		{"non-GitHub SSH", "git@gitlab.com:owner/repo.git", ""},
		{"non-GitHub HTTPS", "https://gitlab.com/owner/repo.git", ""},
		{"malformed", "not-a-url", ""},
		{"empty", "", ""},
		{"with trailing newline", "git@github.com:owner/repo.git\n", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitHubRepo(tt.input)
			if got != tt.expected {
				t.Errorf("parseGitHubRepo(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestInitScaffoldConfigLoads(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".xylem.yml")

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	captureStdout(func() {
		if err := cmdInit(configPath, false); err != nil {
			t.Fatalf("cmdInit failed: %v", err)
		}
	})

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("scaffold config failed to load: %v", err)
	}
	if len(cfg.Sources) == 0 {
		t.Error("expected at least one source in scaffold config")
	}
}

func TestInitRespectsConfigFlag(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom.yml")

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", customPath, "init"})

	captureStdout(func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("init with --config failed: %v", err)
		}
	})

	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("custom config not created at %s: %v", customPath, err)
	}

	// Default path should NOT exist
	if _, err := os.Stat(filepath.Join(dir, ".xylem.yml")); err == nil {
		t.Error(".xylem.yml was created despite --config pointing elsewhere")
	}
}

func TestInitCobraBypassesPersistentPreRunE(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	os.Chdir(dir) //nolint:errcheck
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init"})

	out := captureStdout(func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("init should not fail due to PersistentPreRunE: %v", err)
		}
	})

	if !strings.Contains(out, "Next steps") {
		t.Errorf("expected init output, got: %s", out)
	}
}
