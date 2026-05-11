// Package shim implements the shared runtime for the gh and git PATH shims.
// Each shim resolves the current working directory to a configured GitHub
// App, ensures a non-stale installation token (minting fresh if needed),
// applies tool-specific env/file injections, and execs the real binary.
//
// When no app matches the CWD (or the config is missing), the shim execs
// the real binary unmodified — no token injection. This means the shims
// are safe to install globally on PATH; they only activate inside mapped
// directories.
//
// Properties this layer guarantees:
//
//   - Per-call freshness: every invocation runs session.EnsureToken, which
//     re-mints when the cached token is within the 5-minute refresh window.
//     Tools spawned by Claude Code's Bash tool therefore never see a stale
//     GH_TOKEN, even hours into a long session.
//   - Pass-through fidelity: unmapped CWDs invoke the real binary with the
//     unmodified parent environment. The shim must not silently change
//     auth behavior for repos outside config.
//   - No new long-lived state: token state lives in the existing
//     ~/.cache/claude-github-app/ cache; the git shim's gitconfig file is
//     a single per-app path, rewritten in place each call.
package shim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/realbin"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/session"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

// MintTimeout bounds a single installation-token mint attempt. Matches the
// 45s the claude wrapper uses for parity.
const MintTimeout = 45 * time.Second

// Options is the per-shim contract: the binary it shadows, its argv (minus
// argv[0]), and a function that, given a fresh CacheEntry, returns the env
// injections to apply on exec.
type Options struct {
	// RealName is the bare filename of the binary being shadowed
	// ("gh", "git"). Used by realbin.ResolveByName.
	RealName string

	// Args is os.Args[1:] from the calling shim's main.
	Args []string

	// SelfPath is os.Executable() of the calling shim. Empty values fall
	// through to the soft-fail policy in realbin.
	SelfPath string

	// Inject is called only when a fresh token was obtained. It returns
	// the map of env vars to apply on top of the parent env. The shim
	// runtime takes care of merging.
	//
	// If Inject returns a non-nil cleanup, the runtime runs it inline
	// BEFORE exec (not via defer — real syscall.Exec never returns, so
	// deferred functions never fire). Cleanups are for resources that
	// can be released as soon as env injection is done; anything the
	// child process must read after exec needs to survive the cleanup.
	Inject func(entry *token.CacheEntry, app *config.App) (env map[string]string, cleanup func(), err error)
}

// Run is the entry point. It returns an int suitable for os.Exit.
// On exec the call replaces the current process; on any failure to reach
// exec the function returns a non-zero code and the caller logs.
func Run(opts Options) int {
	if opts.RealName == "" {
		fatal("shim: RealName is required")
	}
	if opts.Inject == nil {
		fatal("shim: Inject is required")
	}

	// Resolve real binary first so PATH problems abort early with a useful
	// error rather than burning a GitHub mint roundtrip.
	real, err := realbin.ResolveByName(opts.RealName, opts.SelfPath)
	if err != nil {
		fatal("real %s not found: %v", opts.RealName, err)
	}

	app, _, matchErr := resolveApp()
	if matchErr != nil {
		// Config corruption: refuse, don't silently fall through to the
		// real binary with potentially stale wrapper-injected auth.
		fatal("config error: %v", matchErr)
	}

	env := os.Environ()

	if app != nil {
		ctx, cancel := context.WithTimeout(context.Background(), MintTimeout)
		defer cancel()

		entry, err := session.EnsureToken(ctx, app)
		if err != nil {
			fatal("token mint failed for app %q: %v", app.Name, err)
		}

		inj, cleanup, err := opts.Inject(entry, app)
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			fatal("inject env for %s: %v", opts.RealName, err)
		}

		env = applyInjections(env, inj)

		// Cleanup must run BEFORE exec, not via defer — real syscall.Exec
		// replaces the process and never returns, so deferred functions
		// never fire. Inject-returned cleanups are expected to be
		// short-lived (write-and-close), so doing them inline is safe.
		if cleanup != nil {
			cleanup()
		}
	}

	// Hand off via exec so the child sees the right $0 and inherits stdio.
	if err := syscallExec(real, append([]string{real}, opts.Args...), env); err != nil {
		fatal("exec %s: %v", real, err)
	}
	// Unreachable.
	return 0
}

// resolveApp returns the App mapped to the current working directory, or
// (nil, nil) if no app applies (no config, no mapping, no default).
func resolveApp() (*config.App, *config.Config, error) {
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cfg, fmt.Errorf("getwd: %w", err)
	}
	app := cfg.Match(filepath.Clean(cwd))
	return app, cfg, nil
}

// applyInjections overrides matching env entries and appends new ones.
// Exported only via the runtime; mirrors the wrapper's identical helper
// rather than depending on it to keep the shim package import-light.
func applyInjections(env []string, injections map[string]string) []string {
	if len(injections) == 0 {
		return env
	}
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

// fatalExit is the bail-out hook. Default implementation logs to stderr +
// status.log and calls os.Exit(1). Tests override to capture the message
// and break out of the run via panic with a known sentinel value.
var fatalExit = func(msg string) {
	_ = token.AppendStatusLog("shim ABORT — " + msg)
	fmt.Fprintln(os.Stderr, "claude-github-app: ABORT — "+msg)
	os.Exit(1)
}

func fatal(format string, args ...any) {
	fatalExit(fmt.Sprintf(format, args...))
}

// ErrNoApp is the sentinel returned by callers' Inject when, after
// inspection, they decide no injection should be performed even though
// CWD matched. Currently unused — kept for future shim behaviors that may
// want to opt out conditionally.
var ErrNoApp = errors.New("no app for this invocation")
