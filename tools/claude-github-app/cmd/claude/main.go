// Command claude is the wrapper entrypoint. It shadows the real Claude Code
// CLI on PATH, resolves the current working directory to a configured GitHub
// App, mints an installation access token, and execs the real claude binary
// with an isolated GH_CONFIG_DIR and GIT_CONFIG_GLOBAL.
//
// All flags are passed through verbatim; the wrapper has no flags of its own.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/launcher"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/realbin"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/session"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

const stderrPrefix = "claude-github-app: "

func main() {
	// Refuse to run under sudo (out of trust boundary per Threat model).
	if os.Geteuid() != os.Getuid() {
		failHard("running under sudo is not supported; invoke as the unprivileged user")
	}

	launcher.SweepStaleTempDirs()

	cfg, cfgErr := loadConfig()

	var (
		app     *config.App
		entry   *token.CacheEntry
		cleanup []func()
		env     = os.Environ()
		status  string
	)

	defer func() {
		runCleanups(cleanup)
	}()

	switch {
	case cfgErr != nil && os.IsNotExist(cfgErr):
		status = "no github app auth'd"
	case cfgErr != nil:
		failHard("config error: %v", cfgErr)
	default:
		cwd, err := os.Getwd()
		if err != nil {
			failHard("getwd: %v", err)
		}
		app = cfg.Match(filepath.Clean(cwd))
		if app == nil {
			status = "no github app auth'd"
		}
	}

	if app != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		e, err := session.EnsureToken(ctx, app)
		if err != nil {
			var pkErr *token.PrivateKeyPermsError
			if errors.As(err, &pkErr) {
				failHard("%v", err)
			}
			failHard("token mint failed for app %q: %v", app.Name, err)
		}
		entry = e

		gitOpts := launcher.GitConfigOpts{Token: entry.Token}
		if entry.BotUserID != 0 && entry.BotLogin != "" {
			gitOpts.BotName = entry.BotLogin
			gitOpts.BotEmail = token.BotCommitEmail(&token.BotUser{ID: entry.BotUserID, Login: entry.BotLogin})
			status = fmt.Sprintf("using github app '%s' for PR + commit identity (installation %d, expires %s)",
				app.Name, app.InstallationID, entry.ExpiresAt.UTC().Format("15:04:05 UTC"))
		} else {
			status = fmt.Sprintf("using github app '%s' for PR identity only — commit author falls back to git default (installation %d, expires %s)",
				app.Name, app.InstallationID, entry.ExpiresAt.UTC().Format("15:04:05 UTC"))
		}

		ghDir, ghClean, err := launcher.TempGHConfigDir()
		if err != nil {
			failHard("create GH_CONFIG_DIR: %v", err)
		}
		cleanup = append(cleanup, ghClean)

		gitCfgPath, gitClean, err := launcher.TempGitConfig(gitOpts)
		if err != nil {
			failHard("create GIT_CONFIG_GLOBAL: %v", err)
		}
		cleanup = append(cleanup, gitClean)

		env = applyInjections(env, map[string]string{
			"GH_TOKEN":              entry.Token,
			"GITHUB_TOKEN":          entry.Token,
			"GH_CONFIG_DIR":         ghDir,
			"GH_PROMPT_DISABLED":    "1",
			"GH_NO_UPDATE_NOTIFIER": "1",
			"GIT_CONFIG_GLOBAL":     gitCfgPath,
			"GIT_CONFIG_NOSYSTEM":   "1",
			"GIT_TERMINAL_PROMPT":   "0",
		})

		// Record where this session's mutable env files live so claude-github-app
		// git-config-refresh can rewrite them in place mid-session.
		_ = writeLastSession(app.Name, ghDir, gitCfgPath)
	}

	// Emit the status line (always log, conditionally print).
	printStatus(status)

	// Resolve real claude.
	selfPath, _ := os.Executable()
	claudeBin := ""
	if cfg != nil {
		claudeBin = cfg.ClaudeBinary
	}
	real, err := realbin.Resolve(claudeBin, selfPath)
	if err != nil {
		failExit(127, "real claude not found: %v", err)
	}

	code, err := launcher.Run(launcher.RunOpts{
		RealClaude: real,
		Args:       os.Args[1:],
		Env:        env,
	})
	if err != nil {
		failExit(code, "%v", err)
	}
	os.Exit(code)
}

func loadConfig() (*config.Config, error) {
	return config.Load(config.DefaultConfigPath)
}

func applyInjections(env []string, injections map[string]string) []string {
	// Replace existing entries by key, append new ones.
	out := make([]string, 0, len(env)+len(injections))
	seen := map[string]bool{}
	for _, e := range env {
		i := strings.IndexByte(e, '=')
		if i < 0 {
			out = append(out, e)
			continue
		}
		key := e[:i]
		if v, ok := injections[key]; ok {
			out = append(out, key+"="+v)
			seen[key] = true
		} else {
			out = append(out, e)
		}
	}
	for k, v := range injections {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func runCleanups(fs []func()) {
	for _, f := range fs {
		f()
	}
}

func printStatus(line string) {
	full := stderrPrefix + line
	_ = token.AppendStatusLog(line)
	if shouldPrintToStderr() {
		fmt.Fprintln(os.Stderr, full)
	}
}

// shouldPrintToStderr enforces "only when stderr is a TTY and we're not
// nested in another Claude Code session" per the plan.
func shouldPrintToStderr() bool {
	if os.Getenv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return false
	}
	info, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func writeLastSession(appName, ghDir, gitCfgPath string) error {
	dir, err := token.CacheDir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, "last-session.json")
	body := fmt.Sprintf(`{"app":%q,"gh_config_dir":%q,"git_config_global":%q,"pid":%d,"ts":%q}`,
		appName, ghDir, gitCfgPath, syscall.Getpid(), time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(p, []byte(body), 0o600)
}

func failHard(format string, args ...any) {
	failExit(1, format, args...)
}

func failExit(code int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_ = token.AppendStatusLog("ABORT — " + msg)
	fmt.Fprintln(os.Stderr, stderrPrefix+"ABORT — "+msg)
	os.Exit(code)
}
