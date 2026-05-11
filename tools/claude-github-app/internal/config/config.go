// Package config loads the claude-github-app TOML configuration and resolves
// the current working directory to a configured GitHub App by longest-prefix
// match, with an EvalSymlinks fallback for symlinked working trees.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultConfigPath = "~/.config/claude-github-app/config.toml"

type Config struct {
	ClaudeBinary string    `toml:"claude_binary"`
	DefaultApp   string    `toml:"default_app"`
	Apps         []App     `toml:"apps"`
	Mappings     []Mapping `toml:"mappings"`
}

type App struct {
	Name           string            `toml:"name"`
	ClientID       string            `toml:"client_id"`
	InstallationID int64             `toml:"installation_id"`
	PrivateKeyFile string            `toml:"private_key_file"`
	RepositoryIDs  []int64           `toml:"repository_ids"`
	BotUserID      int64             `toml:"bot_user_id"`
	Permissions    map[string]string `toml:"permissions"`
}

type Mapping struct {
	Path string `toml:"path"`
	App  string `toml:"app"`

	// pathResolved is the EvalSymlinks'd form of Path, populated at load time
	// when it differs from Path. Used in matchLongest as a third attempt so
	// macOS-style /var → /private/var aliases don't defeat prefix matching
	// when the user's mapping is in logical form and the CWD resolves to the
	// physical form (or vice versa).
	pathResolved string
}

// DefaultPermissions is used when an app's [apps.permissions] table is empty.
func DefaultPermissions() map[string]string {
	return map[string]string{
		"contents":      "write",
		"pull_requests": "write",
		"issues":        "write",
		"workflows":     "write",
	}
}

// Load reads and validates a config file. Returns (nil, os.ErrNotExist) when
// the file is missing — callers translate that into the "no github app auth'd"
// status line rather than an abort.
func Load(path string) (*Config, error) {
	expanded, err := expandTilde(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, err
	}
	var c Config
	if _, err := toml.Decode(string(data), &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", expanded, err)
	}
	if err := c.normalize(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", expanded, err)
	}
	return &c, nil
}

func (c *Config) normalize() error {
	if c.ClaudeBinary != "" {
		exp, err := expandTilde(c.ClaudeBinary)
		if err != nil {
			return err
		}
		c.ClaudeBinary = exp
	}

	seenApps := map[string]bool{}
	for i := range c.Apps {
		a := &c.Apps[i]
		if a.Name == "" {
			return fmt.Errorf("apps[%d]: name is required", i)
		}
		if seenApps[a.Name] {
			return fmt.Errorf("apps[%d]: duplicate name %q", i, a.Name)
		}
		seenApps[a.Name] = true
		if a.ClientID == "" {
			return fmt.Errorf("apps[%q]: client_id is required", a.Name)
		}
		if a.InstallationID == 0 {
			return fmt.Errorf("apps[%q]: installation_id is required", a.Name)
		}
		if a.PrivateKeyFile == "" {
			return fmt.Errorf("apps[%q]: private_key_file is required", a.Name)
		}
		exp, err := expandTilde(a.PrivateKeyFile)
		if err != nil {
			return fmt.Errorf("apps[%q]: %w", a.Name, err)
		}
		a.PrivateKeyFile = exp
		if len(a.Permissions) == 0 {
			a.Permissions = DefaultPermissions()
		}
	}

	seenPaths := map[string]int{}
	for i := range c.Mappings {
		m := &c.Mappings[i]
		if m.Path == "" {
			return fmt.Errorf("mappings[%d]: path is required", i)
		}
		if m.App == "" {
			return fmt.Errorf("mappings[%d]: app is required", i)
		}
		if !seenApps[m.App] {
			return fmt.Errorf("mappings[%d]: app %q is not defined under [[apps]]", i, m.App)
		}
		exp, err := expandTilde(m.Path)
		if err != nil {
			return fmt.Errorf("mappings[%d]: %w", i, err)
		}
		m.Path = filepath.Clean(exp)
		if prev, ok := seenPaths[m.Path]; ok {
			return fmt.Errorf("mappings[%d]: duplicate path %q (also at mappings[%d])", i, m.Path, prev)
		}
		seenPaths[m.Path] = i
		if resolved, err := filepath.EvalSymlinks(m.Path); err == nil {
			resolved = filepath.Clean(resolved)
			if resolved != m.Path {
				m.pathResolved = resolved
			}
		}
	}

	if c.DefaultApp != "" && !seenApps[c.DefaultApp] {
		return fmt.Errorf("default_app %q is not defined under [[apps]]", c.DefaultApp)
	}

	return nil
}

// FindApp returns a pointer to the named app or nil.
func (c *Config) FindApp(name string) *App {
	for i := range c.Apps {
		if c.Apps[i].Name == name {
			return &c.Apps[i]
		}
	}
	return nil
}

// Match resolves a working directory to a configured app, applying:
//  1. longest-prefix match on the logical cwd
//  2. on miss, retry against filepath.EvalSymlinks(cwd)
//  3. on miss, fall back to default_app
//
// Returns nil if no match.
func (c *Config) Match(cwd string) *App {
	cwd = filepath.Clean(cwd)
	if app := c.matchLongest(cwd); app != nil {
		return app
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil && resolved != cwd {
		if app := c.matchLongest(filepath.Clean(resolved)); app != nil {
			return app
		}
	}
	if c.DefaultApp != "" {
		return c.FindApp(c.DefaultApp)
	}
	return nil
}

func (c *Config) matchLongest(cwd string) *App {
	type candidate struct {
		idx int
		l   int
	}
	var best []candidate
	for i, m := range c.Mappings {
		switch {
		case isPathPrefix(cwd, m.Path):
			best = append(best, candidate{i, len(m.Path)})
		case m.pathResolved != "" && isPathPrefix(cwd, m.pathResolved):
			best = append(best, candidate{i, len(m.pathResolved)})
		}
	}
	if len(best) == 0 {
		return nil
	}
	sort.SliceStable(best, func(i, j int) bool {
		if best[i].l != best[j].l {
			return best[i].l > best[j].l
		}
		return best[i].idx < best[j].idx // tie-break by config order
	})
	return c.FindApp(c.Mappings[best[0].idx].App)
}

// isPathPrefix reports whether path is a directory-prefix of cwd:
//
//	"/a/b" is a prefix of "/a/b"     (exact match)
//	"/a/b" is a prefix of "/a/b/c"   (separator after prefix)
//	"/a/b" is NOT a prefix of "/a/bb" (no separator after prefix)
func isPathPrefix(cwd, prefix string) bool {
	if !strings.HasPrefix(cwd, prefix) {
		return false
	}
	if len(cwd) == len(prefix) {
		return true
	}
	return cwd[len(prefix)] == filepath.Separator
}

func expandTilde(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// ErrConfigMissing is reported by callers to distinguish "file not found"
// (silent passthrough) from other Load errors (abort).
var ErrConfigMissing = errors.New("config file does not exist")
