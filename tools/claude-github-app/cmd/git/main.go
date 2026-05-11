// Command git is a PATH shim that shadows the real `git` CLI and refreshes
// the GitHub App installation token before exec'ing the real binary. It
// pairs with the gh shim to defeat mid-session token expiry: every git
// invocation (especially `git push` and `git fetch`) sees a current token,
// regardless of how long the parent claude session has been running.
//
// The shim writes a per-app gitconfig at a stable cache path
// (~/.cache/claude-github-app/<app>-gitconfig) and exports
// GIT_CONFIG_GLOBAL pointing at it. The file is rewritten on every call
// from the same cache that powers the gh shim and the wrapper itself.
//
// Mapping behavior matches the gh shim: unmapped CWDs exec real git with
// the unmodified parent environment. The shim must not change auth
// behavior for repos outside [[mappings]].
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/launcher"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/shim"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

func main() {
	if os.Geteuid() != os.Getuid() {
		_, _ = os.Stderr.WriteString("claude-github-app: git shim refuses to run under sudo\n")
		os.Exit(1)
	}

	selfPath, _ := os.Executable()

	code := shim.Run(shim.Options{
		RealName: "git",
		Args:     os.Args[1:],
		SelfPath: selfPath,
		Inject:   injectGitAuth,
	})
	os.Exit(code)
}

func injectGitAuth(entry *token.CacheEntry, app *config.App) (map[string]string, func(), error) {
	cfgPath, err := gitConfigPath(app.Name)
	if err != nil {
		return nil, nil, err
	}
	opts := launcher.GitConfigOpts{Token: entry.Token}
	if entry.BotUserID != 0 && entry.BotLogin != "" {
		opts.BotName = entry.BotLogin
		opts.BotEmail = token.BotCommitEmail(&token.BotUser{ID: entry.BotUserID, Login: entry.BotLogin})
	}
	if err := launcher.WriteGitConfigAtomic(cfgPath, opts); err != nil {
		return nil, nil, fmt.Errorf("write gitconfig %s: %w", cfgPath, err)
	}
	env := map[string]string{
		"GIT_CONFIG_GLOBAL":   cfgPath,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	}
	return env, nil, nil
}

// gitConfigPath returns ~/.cache/claude-github-app/<app>-gitconfig.
// Lives alongside the token cache (~/.cache/claude-github-app/<app>.json)
// so a single sweep / removal cleans up everything for an app.
func gitConfigPath(appName string) (string, error) {
	dir, err := token.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName+"-gitconfig"), nil
}
