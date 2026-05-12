package launcher

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitConfigOpts controls the contents of the temp GIT_CONFIG_GLOBAL file.
type GitConfigOpts struct {
	// Token is required. It is the GitHub App installation token used as the
	// password in HTTP Basic auth (username "x-access-token") for the
	// http.extraHeader entry. GitHub's git smart-HTTP transport rejects
	// Bearer auth on /info/refs and /git-{upload,receive}-pack with HTTP 401,
	// even though the same token works as Bearer against api.github.com.
	Token    string
	BotName  string // optional — set [user] block when both BotName and BotEmail are set
	BotEmail string
}

// RenderGitConfig returns the rendered gitconfig contents for opts.
// Returns an error only when the token is empty (the one invariant we
// refuse to write a malformed config for). Pure: no file system side
// effects.
func RenderGitConfig(opts GitConfigOpts) (string, error) {
	if opts.Token == "" {
		return "", fmt.Errorf("git config: token is required")
	}
	var b strings.Builder
	if opts.BotName != "" && opts.BotEmail != "" {
		fmt.Fprintf(&b, "[user]\n\tname = %s\n\temail = %s\n", opts.BotName, opts.BotEmail)
	}
	// Clear any inherited credential helper so git doesn't reach for the keychain.
	fmt.Fprintf(&b, "[credential]\n\thelper =\n")
	// First extraHeader empty value clears any inherited header chain (defensive
	// — git concatenates extraHeader multivar values, so resetting first prevents
	// any system-level header from being appended alongside ours). Second sets
	// our auth: Basic base64("x-access-token:<token>"). Bearer is rejected by
	// git's smart-HTTP transport with 401 — see GitConfigOpts.Token doc.
	creds := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + opts.Token))
	fmt.Fprintf(&b, "[http]\n\textraHeader =\n\textraHeader = Authorization: Basic %s\n", creds)
	return b.String(), nil
}

// WriteGitConfigAtomic writes the rendered config to dst at mode 0600 via a
// same-directory temp file + rename, so concurrent readers (e.g. git fork
// in another shim invocation) never see a half-written file. The parent
// directory is created at 0700 if missing.
func WriteGitConfigAtomic(dst string, opts GitConfigOpts) error {
	contents, err := RenderGitConfig(opts)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create git config dir: %w", err)
	}
	// Best-effort tighten in case the dir pre-existed with broader perms.
	_ = os.Chmod(dir, 0o700)

	f, err := os.CreateTemp(dir, ".gitconfig-*")
	if err != nil {
		return fmt.Errorf("create temp git config: %w", err)
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if _, err := f.WriteString(contents); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write git config: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("chmod git config: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		cleanup()
		return fmt.Errorf("rename git config: %w", err)
	}
	return nil
}

// TempGitConfig writes a temp gitconfig file at mode 0600 containing the
// extraHeader auth, an empty credential.helper, and optionally a [user] block
// for bot commit identity. Returns the file path and a cleanup func that
// removes the enclosing directory.
//
// The file path is suitable for use as GIT_CONFIG_GLOBAL.
func TempGitConfig(opts GitConfigOpts) (string, func(), error) {
	contents, err := RenderGitConfig(opts)
	if err != nil {
		return "", nil, err
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

	cfgPath := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(cfgPath, []byte(contents), 0o600); err != nil {
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
