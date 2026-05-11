// Command claude-github-app is the management binary. Subcommands:
//
//	token [--app NAME]       Print a fresh installation token to stdout. The
//	                         token is the cached value if not stale, otherwise
//	                         a freshly minted one. Useful inside a Claude
//	                         session: `GH_TOKEN=$(claude-github-app token) gh ...`
//
//	git-config-refresh       Rewrite the active session's GIT_CONFIG_GLOBAL
//	                         file in place with a fresh Bearer token. Reads
//	                         the path from ~/.cache/claude-github-app/last-session.json.
//
//	status                   Print a summary of cached tokens per app.
//
// All subcommands refuse to run under sudo (Properties §A trust boundary).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/launcher"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/session"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

func main() {
	if os.Geteuid() != os.Getuid() {
		fatal("running under sudo is not supported")
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "token":
		os.Exit(cmdToken(os.Args[2:]))
	case "git-config-refresh":
		os.Exit(cmdGitConfigRefresh(os.Args[2:]))
	case "status":
		os.Exit(cmdStatus(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `claude-github-app: manage GitHub App tokens for the claude wrapper

Usage:
  claude-github-app token [--app NAME]    Print a fresh token to stdout
  claude-github-app git-config-refresh    Rewrite active session's GIT_CONFIG_GLOBAL
  claude-github-app status                Show cached token state per app

The app is selected by:
  1. --app flag (or 'app' positional arg) if given
  2. Current working directory mapping in ~/.config/claude-github-app/config.toml
  3. default_app from config
`)
}

func loadConfig() *config.Config {
	c, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		fatal("load config: %v", err)
	}
	return c
}

// loadConfigOrEmpty is like loadConfig but treats a missing file as an empty
// config (returns &Config{} with no apps/mappings) rather than fataling. Used
// by `status` so the command can report "no config" without exiting 1.
func loadConfigOrEmpty() *config.Config {
	c, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &config.Config{}
		}
		fatal("load config: %v", err)
	}
	return c
}

func pickApp(c *config.Config, override string) *config.App {
	if override != "" {
		app := c.FindApp(override)
		if app == nil {
			fatal("app %q not found in config", override)
		}
		return app
	}
	cwd, err := os.Getwd()
	if err != nil {
		fatal("getwd: %v", err)
	}
	app := c.Match(filepath.Clean(cwd))
	if app == nil {
		fatal("no app mapped to %s (and no default_app set); use --app NAME", cwd)
	}
	return app
}

func cmdToken(args []string) int {
	fs := flag.NewFlagSet("token", flag.ExitOnError)
	appFlag := fs.String("app", "", "app name (overrides CWD mapping)")
	_ = fs.Parse(args)

	c := loadConfig()
	app := pickApp(c, *appFlag)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	entry, err := session.EnsureToken(ctx, app)
	if err != nil {
		fatal("%v", err)
	}
	// Print ONLY the token, no trailing newline noise — callers use $(...) capture.
	fmt.Println(entry.Token)
	return 0
}

func cmdGitConfigRefresh(args []string) int {
	fs := flag.NewFlagSet("git-config-refresh", flag.ExitOnError)
	appFlag := fs.String("app", "", "app name (overrides last-session record)")
	_ = fs.Parse(args)

	last, err := readLastSession()
	if err != nil {
		fatal("no active session found (is claude running?): %v", err)
	}
	c := loadConfig()
	appName := *appFlag
	if appName == "" {
		appName = last.App
	}
	app := c.FindApp(appName)
	if app == nil {
		fatal("app %q not found in config", appName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	entry, err := session.EnsureToken(ctx, app)
	if err != nil {
		fatal("%v", err)
	}

	gitOpts := launcher.GitConfigOpts{Token: entry.Token}
	if entry.BotUserID != 0 && entry.BotLogin != "" {
		gitOpts.BotName = entry.BotLogin
		gitOpts.BotEmail = token.BotCommitEmail(&token.BotUser{ID: entry.BotUserID, Login: entry.BotLogin})
	}
	// Rewrite the in-place file. We can't use launcher.TempGitConfig because
	// that creates a fresh temp dir; instead, write directly to the existing
	// path with 0600 mode preserved.
	if err := rewriteGitConfig(last.GitConfigGlobal, gitOpts); err != nil {
		fatal("rewrite git config: %v", err)
	}
	fmt.Printf("refreshed %s; new expiry %s\n", last.GitConfigGlobal, entry.ExpiresAt.UTC().Format(time.RFC3339))
	return 0
}

func cmdStatus(args []string) int {
	_ = args
	c := loadConfigOrEmpty()
	if len(c.Apps) == 0 {
		fmt.Println("(no apps configured — drop a config at ~/.config/claude-github-app/config.toml)")
		return 0
	}
	dir, _ := token.CacheDir()
	for _, app := range c.Apps {
		entry, err := token.ReadCache(app.Name)
		switch {
		case err != nil:
			fmt.Printf("%-30s  no cache\n", app.Name)
		case entry.Stale():
			fmt.Printf("%-30s  STALE   expires %s\n", app.Name, entry.ExpiresAt.UTC().Format(time.RFC3339))
		default:
			remaining := time.Until(entry.ExpiresAt).Truncate(time.Second)
			fmt.Printf("%-30s  fresh   expires %s (%s left)\n", app.Name, entry.ExpiresAt.UTC().Format(time.RFC3339), remaining)
		}
	}
	fmt.Printf("\ncache dir: %s\n", dir)
	return 0
}

type lastSession struct {
	App             string `json:"app"`
	GHConfigDir     string `json:"gh_config_dir"`
	GitConfigGlobal string `json:"git_config_global"`
}

func readLastSession() (*lastSession, error) {
	dir, err := token.CacheDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "last-session.json"))
	if err != nil {
		return nil, err
	}
	var ls lastSession
	if err := json.Unmarshal(data, &ls); err != nil {
		return nil, err
	}
	return &ls, nil
}

func rewriteGitConfig(path string, opts launcher.GitConfigOpts) error {
	// Build the same contents launcher.TempGitConfig builds, but write to the
	// existing path so the child claude's $GIT_CONFIG_GLOBAL keeps pointing at
	// it.
	if path == "" {
		return fmt.Errorf("no git config path recorded")
	}
	tmp, cleanup, err := launcher.TempGitConfig(opts)
	if err != nil {
		return err
	}
	defer cleanup()
	data, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "claude-github-app: "+format+"\n", args...)
	os.Exit(1)
}
