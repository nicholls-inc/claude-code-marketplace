package config

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const minTimeout = 30 * time.Second

const defaultClaudeTemplate = "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"

type Config struct {
	Repo        string          `yaml:"repo"`
	Tasks       map[string]Task `yaml:"tasks"`
	Concurrency int             `yaml:"concurrency"`
	MaxTurns    int             `yaml:"max_turns"`
	Timeout     string          `yaml:"timeout"`
	StateDir    string          `yaml:"state_dir"`
	Exclude     []string        `yaml:"exclude"`
	Claude      ClaudeConfig    `yaml:"claude"`
}

type Task struct {
	Labels []string `yaml:"labels"`
	Skill  string   `yaml:"skill"`
}

type ClaudeConfig struct {
	Command  string `yaml:"command"`
	Template string `yaml:"template"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg := &Config{
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		StateDir:    ".pit-crew",
		Exclude:     []string{"wontfix", "duplicate", "in-progress", "no-bot"},
		Claude: ClaudeConfig{
			Command:  "claude",
			Template: defaultClaudeTemplate,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	repo := strings.TrimSpace(c.Repo)
	if repo == "" {
		return fmt.Errorf("repo is required")
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("repo must be in owner/name format")
	}

	if len(c.Tasks) == 0 {
		return fmt.Errorf("tasks: at least one task is required")
	}

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

	if _, err := template.New("cfg").Parse(c.Claude.Template); err != nil {
		return fmt.Errorf("claude.template is not a valid Go template: %w", err)
	}

	for name, task := range c.Tasks {
		if len(task.Labels) == 0 {
			return fmt.Errorf("task %q must include at least one labels entry", name)
		}

		if strings.TrimSpace(task.Skill) == "" {
			return fmt.Errorf("task %q must include a skill", name)
		}
	}

	return nil
}
