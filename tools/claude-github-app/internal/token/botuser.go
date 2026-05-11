package token

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// BotUser is the public-API view of an App's bot identity.
type BotUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

// ResolveBotUser fetches GET /users/<app-slug>[bot] using an installation
// token. Returns the numeric ID and the canonical login (e.g. "my-app[bot]").
// The result is cached alongside the installation token; the wrapper does NOT
// refetch this on token refresh.
func ResolveBotUser(ctx context.Context, installationToken, appSlug string) (*BotUser, error) {
	path := url.PathEscape(appSlug + "[bot]")
	u := fmt.Sprintf("%s/users/%s", APIBase, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lookup bot user %s[bot]: %w", appSlug, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bot user lookup failed for %s[bot]: %d %s: %s",
			appSlug, resp.StatusCode, http.StatusText(resp.StatusCode), string(b))
	}
	var bu BotUser
	if err := json.NewDecoder(resp.Body).Decode(&bu); err != nil {
		return nil, fmt.Errorf("decode bot user response: %w", err)
	}
	if bu.ID == 0 || bu.Login == "" {
		return nil, fmt.Errorf("bot user response missing id/login: %+v", bu)
	}
	return &bu, nil
}

// BotCommitEmail builds the noreply email that GitHub uses to attribute
// commits to a bot user — `<id>+<login>@users.noreply.github.com`.
func BotCommitEmail(bu *BotUser) string {
	return fmt.Sprintf("%d+%s@users.noreply.github.com", bu.ID, bu.Login)
}
