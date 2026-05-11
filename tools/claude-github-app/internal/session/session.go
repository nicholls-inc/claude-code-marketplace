// Package session encapsulates the "ensure I have a fresh installation token
// for app X" workflow shared between the wrapper (cmd/claude) and the
// management binary (cmd/claude-github-app).
package session

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

// EnsureToken returns a non-stale cache entry for an app, minting a new token
// (and persisting it) if the cache is missing or expiring soon. On first mint
// it also resolves the bot user; subsequent refreshes reuse the bot identity
// (per Properties §B commit-author invariant).
//
// If the app has BotUserID set in config, that overrides any lookup.
func EnsureToken(ctx context.Context, app *config.App) (*token.CacheEntry, error) {
	if app == nil {
		return nil, errors.New("app is nil")
	}

	if cached, err := token.ReadCache(app.Name); err == nil && !cached.Stale() {
		// Backfill bot identity from explicit config if it's now set but not cached.
		if cached.BotUserID == 0 && app.BotUserID != 0 {
			cached.BotUserID = app.BotUserID
			cached.BotLogin = app.Name + "[bot]"
			_ = token.WriteCache(app.Name, cached)
		}
		return cached, nil
	}

	pemBytes, err := token.ReadPrivateKey(app.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	appJWT, err := token.BuildAppJWT(app.ClientID, pemBytes)
	if err != nil {
		return nil, fmt.Errorf("build jwt for app %q: %w", app.Name, err)
	}

	tok, err := token.MintInstallationToken(ctx, appJWT, app.InstallationID, app.Permissions, app.RepositoryIDs)
	if err != nil {
		return nil, err
	}

	entry := &token.CacheEntry{
		Token:     tok.Token,
		ExpiresAt: tok.ExpiresAt,
	}

	// Preserve any previously known bot identity across re-mints.
	if prev, err := token.ReadCache(app.Name); err == nil {
		entry.BotUserID = prev.BotUserID
		entry.BotLogin = prev.BotLogin
	}

	if app.BotUserID != 0 {
		entry.BotUserID = app.BotUserID
		entry.BotLogin = app.Name + "[bot]"
	}
	if entry.BotUserID == 0 {
		bu, err := token.ResolveBotUser(ctx, tok.Token, app.Name)
		if err == nil {
			entry.BotUserID = bu.ID
			entry.BotLogin = bu.Login
		} else {
			// Non-fatal: caller will degrade the commit-author invariant only.
			fmt.Fprintf(os.Stderr, "claude-github-app: bot user lookup failed for %s: %v\n", app.Name, err)
		}
	}

	if err := token.WriteCache(app.Name, entry); err != nil {
		return nil, fmt.Errorf("write token cache: %w", err)
	}
	return entry, nil
}
