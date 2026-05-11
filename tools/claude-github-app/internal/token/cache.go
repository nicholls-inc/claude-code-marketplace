package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RefreshWindow is how close to expiry we tolerate before re-minting.
const RefreshWindow = 5 * time.Minute

// CacheEntry is the JSON shape persisted at ~/.cache/claude-github-app/<app>.json.
// Includes the bot identity so it survives token refreshes without re-lookup.
type CacheEntry struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	BotUserID int64     `json:"bot_user_id,omitempty"`
	BotLogin  string    `json:"bot_login,omitempty"`
}

func (e *CacheEntry) Stale() bool {
	return time.Until(e.ExpiresAt) < RefreshWindow
}

// CacheDir resolves to ~/.cache/claude-github-app, creating it at 0700 if missing.
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "claude-github-app")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// Tighten in case it was pre-created with broader perms.
	_ = os.Chmod(dir, 0o700)
	return dir, nil
}

// ReadCache returns the cached entry for an app, or os.ErrNotExist if absent.
func ReadCache(appName string) (*CacheEntry, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(dir, appName+".json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var e CacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("decode cache %s: %w", p, err)
	}
	return &e, nil
}

// WriteCache atomically writes an entry at mode 0600. Uses os.CreateTemp to
// avoid collisions with parallel `claude` launches.
func WriteCache(appName string, e *CacheEntry) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "tok-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return err
	}
	final := filepath.Join(dir, appName+".json")
	if err := os.Rename(tmp, final); err != nil {
		cleanup()
		return err
	}
	return nil
}

// AppendStatusLog appends a single timestamped line to status.log at 0600.
// Errors are non-fatal — the caller logs them to stderr and continues.
func AppendStatusLog(line string) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, "status.log")
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
	return err
}

// ErrNoCache is a sentinel for "cache file not present".
var ErrNoCache = errors.New("no cache entry")
