package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const minTimeout = 30 * time.Second

type Config struct {
	Repo          string                  `yaml:"repo,omitempty"`
	Sources       map[string]SourceConfig `yaml:"sources,omitempty"`
	Tasks         map[string]Task         `yaml:"tasks,omitempty"`
	Concurrency   int                     `yaml:"concurrency"`
	MaxTurns      int                     `yaml:"max_turns"`
	Timeout       string                  `yaml:"timeout"`
	StateDir      string                  `yaml:"state_dir"`
	Exclude       []string                `yaml:"exclude,omitempty"`
	DefaultBranch string                  `yaml:"default_branch,omitempty"`
	CleanupAfter  string                  `yaml:"cleanup_after,omitempty"`
	Claude        ClaudeConfig            `yaml:"claude"`
	Daemon        DaemonConfig            `yaml:"daemon,omitempty"`
}

type SourceConfig struct {
	Type    string          `yaml:"type"`
	Repo    string          `yaml:"repo,omitempty"`
	Exclude []string        `yaml:"exclude,omitempty"`
	Tasks   map[string]Task `yaml:"tasks,omitempty"`
}

type Task struct {
	Labels []string `yaml:"labels,omitempty"`
	Skill  string   `yaml:"skill"`
}

type ClaudeConfig struct {
	Command  string            `yaml:"command"`
	Flags    string            `yaml:"flags,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	// Template is kept for deserialization so we can detect and reject it.
	Template     string   `yaml:"template,omitempty"`
	AllowedTools []string `yaml:"allowed_tools,omitempty"`
}

type DaemonConfig struct {
	ScanInterval  string `yaml:"scan_interval,omitempty"`
	DrainInterval string `yaml:"drain_interval,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg := &Config{
		Concurrency:  2,
		MaxTurns:     50,
		Timeout:      "30m",
		StateDir:     ".xylem",
		CleanupAfter: "168h",
		Exclude:      []string{"wontfix", "duplicate", "in-progress", "no-bot"},
		Claude: ClaudeConfig{
			Command: "claude",
		},
		Daemon: DaemonConfig{
			ScanInterval:  "60s",
			DrainInterval: "30s",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	cfg.normalize()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// normalize migrates legacy top-level Repo/Tasks/Exclude into the Sources map.
func (c *Config) normalize() {
	if c.Repo != "" && len(c.Sources) == 0 && len(c.Tasks) > 0 {
		exclude := c.Exclude
		c.Sources = map[string]SourceConfig{
			"github": {
				Type:    "github",
				Repo:    c.Repo,
				Exclude: exclude,
				Tasks:   c.Tasks,
			},
		}
	}
}

func (c *Config) Validate() error {
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	if c.MaxTurns <= 0 {
		return fmt.Errorf("max_turns must be greater than 0")
	}

	dur, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return fmt.Errorf("timeout must be a valid duration: %w", err)
	}
	if dur < minTimeout {
		return fmt.Errorf("timeout must be at least %s", minTimeout)
	}

	if c.CleanupAfter != "" {
		if _, err := time.ParseDuration(c.CleanupAfter); err != nil {
			return fmt.Errorf("cleanup_after must be a valid duration: %w", err)
		}
	}

	if c.Claude.Template != "" {
		return fmt.Errorf("claude.template is no longer supported; migrate to phase-based skills in .xylem/skills/")
	}

	if len(c.Claude.AllowedTools) > 0 {
		return fmt.Errorf("claude.allowed_tools is no longer supported; use allowed_tools in skill phase definitions")
	}

	if strings.Contains(c.Claude.Flags, "--bare") {
		if c.Claude.Env == nil || c.Claude.Env["ANTHROPIC_API_KEY"] == "" {
			return fmt.Errorf("--bare requires ANTHROPIC_API_KEY in claude.env")
		}
	}

	if c.Daemon.ScanInterval != "" {
		if _, err := time.ParseDuration(c.Daemon.ScanInterval); err != nil {
			return fmt.Errorf("daemon.scan_interval must be a valid duration: %w", err)
		}
	}
	if c.Daemon.DrainInterval != "" {
		if _, err := time.ParseDuration(c.Daemon.DrainInterval); err != nil {
			return fmt.Errorf("daemon.drain_interval must be a valid duration: %w", err)
		}
	}

	// Validate sources
	for name, src := range c.Sources {
		switch src.Type {
		case "github":
			if err := validateGitHubSource(name, src); err != nil {
				return err
			}
		case "":
			return fmt.Errorf("source %q must specify a type", name)
		}
	}

	// Legacy validation: if top-level Repo is set without Sources, validate it
	if c.Repo != "" && len(c.Sources) == 0 {
		parts := strings.Split(c.Repo, "/")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("repo must be in owner/name format")
		}
		if len(c.Tasks) == 0 {
			return fmt.Errorf("tasks: at least one task is required")
		}
		for tname, task := range c.Tasks {
			if len(task.Labels) == 0 {
				return fmt.Errorf("task %q must include at least one labels entry", tname)
			}
			if strings.TrimSpace(task.Skill) == "" {
				return fmt.Errorf("task %q must include a skill", tname)
			}
		}
	}

	return nil
}

// CleanupAfterDuration returns the parsed cleanup_after duration, defaulting to
// 168h (7 days) if the field is empty or unparseable.
func (c *Config) CleanupAfterDuration() time.Duration {
	if c.CleanupAfter == "" {
		return 168 * time.Hour
	}
	d, err := time.ParseDuration(c.CleanupAfter)
	if err != nil {
		return 168 * time.Hour
	}
	return d
}

func validateGitHubSource(name string, src SourceConfig) error {
	repo := strings.TrimSpace(src.Repo)
	if repo == "" {
		return fmt.Errorf("source %q (github): repo is required", name)
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("source %q (github): repo must be in owner/name format", name)
	}
	if len(src.Tasks) == 0 {
		return fmt.Errorf("source %q (github): at least one task is required", name)
	}
	for tname, task := range src.Tasks {
		if len(task.Labels) == 0 {
			return fmt.Errorf("source %q task %q: must include at least one labels entry", name, tname)
		}
		if strings.TrimSpace(task.Skill) == "" {
			return fmt.Errorf("source %q task %q: must include a skill", name, tname)
		}
	}
	return nil
}
