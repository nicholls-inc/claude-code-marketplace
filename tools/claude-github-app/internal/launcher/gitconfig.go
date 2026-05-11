package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitConfigOpts controls the contents of the temp GIT_CONFIG_GLOBAL file.
type GitConfigOpts struct {
	Token    string // required — Bearer token for http.extraHeader
	BotName  string // optional — set [user] block when both BotName and BotEmail are set
	BotEmail string
}

// TempGitConfig writes a temp gitconfig file at mode 0600 containing the
// extraHeader auth, an empty credential.helper, and optionally a [user] block
// for bot commit identity. Returns the file path and a cleanup func that
// removes the enclosing directory.
//
// The file path is suitable for use as GIT_CONFIG_GLOBAL.
func TempGitConfig(opts GitConfigOpts) (string, func(), error) {
	if opts.Token == "" {
		return "", nil, fmt.Errorf("git config: token is required")
	}
	dir, err := os.MkdirTemp("", "claude-github-app-*-git")
	if err != nil {
		return "", nil, fmt.Errorf("create git config dir: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("chmod git config dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	var b strings.Builder
	if opts.BotName != "" && opts.BotEmail != "" {
		fmt.Fprintf(&b, "[user]\n\tname = %s\n\temail = %s\n", opts.BotName, opts.BotEmail)
	}
	// Clear any inherited credential helper so git doesn't reach for the keychain.
	fmt.Fprintf(&b, "[credential]\n\thelper =\n")
	// First extraHeader empty value clears any inherited header chain (defensive);
	// second sets ours. git concatenates extraHeader values, so resetting first
	// is the documented pattern.
	fmt.Fprintf(&b, "[http]\n\textraHeader =\n\textraHeader = Authorization: Bearer %s\n", opts.Token)

	cfgPath := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(cfgPath, []byte(b.String()), 0o600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write git config: %w", err)
	}
	// Belt-and-braces chmod in case umask was unusual.
	if err := os.Chmod(cfgPath, 0o600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("chmod git config: %w", err)
	}
	return cfgPath, cleanup, nil
}
