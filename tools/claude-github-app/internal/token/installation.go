package token

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// APIBase is the GitHub REST API root. Overridable for tests.
var APIBase = "https://api.github.com"

// Token is the parsed response from POST /app/installations/{id}/access_tokens.
type Token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MintRetryBudget caps the total time spent retrying transient 5xx responses.
const MintRetryBudget = 30 * time.Second

// MintInstallationToken exchanges an App JWT for a 1-hour installation token.
// On 5xx it retries with exponential jitter up to MintRetryBudget. 4xx is not
// retried; 2xx outside 201 is treated as an unexpected response.
func MintInstallationToken(
	ctx context.Context,
	appJWT string,
	installationID int64,
	permissions map[string]string,
	repositoryIDs []int64,
) (*Token, error) {
	bodyMap := map[string]interface{}{}
	if len(permissions) > 0 {
		bodyMap["permissions"] = permissions
	}
	if len(repositoryIDs) > 0 {
		bodyMap["repository_ids"] = repositoryIDs
	}
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", APIBase, installationID)
	deadline := time.Now().Add(MintRetryBudget)
	backoff := time.Second

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+appJWT)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusCreated {
			defer resp.Body.Close()
			var t Token
			if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
				return nil, fmt.Errorf("decode token response: %w", err)
			}
			return &t, nil
		}

		// Read body for error reporting / retry decision
		var status int
		var bodyText string
		if resp != nil {
			status = resp.StatusCode
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			bodyText = string(b)
		}

		retryable := err != nil || (status >= 500 && status <= 599)
		if !retryable {
			return nil, fmt.Errorf("token mint failed for installation %d: %d %s: %s",
				installationID, status, http.StatusText(status), bodyText)
		}

		sleep := backoff + time.Duration(rand.Int64N(int64(backoff)))
		if time.Now().Add(sleep).After(deadline) {
			if err != nil {
				return nil, fmt.Errorf("token mint failed for installation %d after %d retries: %w",
					installationID, attempt, err)
			}
			return nil, fmt.Errorf("token mint failed for installation %d after %d retries: last status %d: %s",
				installationID, attempt, status, bodyText)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleep):
		}
		backoff *= 2
	}
}
