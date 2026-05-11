// Command gh is a PATH shim that shadows the real `gh` CLI on PATH and
// transparently re-mints / refreshes the GitHub App installation token
// before exec'ing the real binary. It exists to defeat the "GH_TOKEN is
// frozen in the child once claude is exec'd" problem: every gh invocation
// inside a long-running claude session gets a fresh token regardless of
// how long ago the parent claude was launched.
//
// All flags and args are passed through verbatim — the shim has no flags
// of its own.
//
// When the current working directory does not match any [[mappings]] in
// ~/.config/claude-github-app/config.toml, the shim execs the real gh
// with the unmodified parent environment. This makes the shim safe to
// install globally on PATH.
package main

import (
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/shim"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

func main() {
	// Refuse to run under sudo to preserve the trust boundary on the
	// private key file and token cache (matches Properties §A).
	if os.Geteuid() != os.Getuid() {
		_, _ = os.Stderr.WriteString("claude-github-app: gh shim refuses to run under sudo\n")
		os.Exit(1)
	}

	selfPath, _ := os.Executable()

	code := shim.Run(shim.Options{
		RealName: "gh",
		Args:     os.Args[1:],
		SelfPath: selfPath,
		Inject: func(entry *token.CacheEntry, _ *config.App) (map[string]string, func(), error) {
			// gh reads GH_TOKEN with highest precedence; GITHUB_TOKEN
			// is set for completeness so any gh extensions that read
			// the generic var also see the fresh token.
			//
			// We deliberately do NOT override GH_CONFIG_DIR here.
			// The parent claude wrapper (if active) set it to an
			// isolated temp dir; an outside-the-wrapper invocation
			// inherits the user's default. Either way, our token
			// env vars take precedence over any hosts.yml.
			return map[string]string{
				"GH_TOKEN":              entry.Token,
				"GITHUB_TOKEN":          entry.Token,
				"GH_PROMPT_DISABLED":    "1",
				"GH_NO_UPDATE_NOTIFIER": "1",
			}, nil, nil
		},
	})
	os.Exit(code)
}
