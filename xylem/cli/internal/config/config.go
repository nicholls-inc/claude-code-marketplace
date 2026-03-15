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

const defaultClaudeTemplate = "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}"

// legacyClaudeTemplate is the old default using {{.IssueURL}} for backward compat.
const legacyClaudeTemplate = "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"

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
	Claude        ClaudeConfig            `yaml:"claude"`
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
		StateDir:    ".xylem",
		Exclude:     []string{"wontfix", "duplicate", "in-progress", "no-bot"},
		Claude: ClaudeConfig{
			Command:  "claude",
			Template: defaultClaudeTemplate,
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
		// Upgrade legacy template to use {{.Ref}} if it was the old default
		if c.Claude.Template == legacyClaudeTemplate {
			c.Claude.Template = defaultClaudeTemplate
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

	if _, err := template.New("cfg").Parse(c.Claude.Template); err != nil {
		return fmt.Errorf("claude.template is not a valid Go template: %w", err)
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
