package token

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func mustGenPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func TestBuildAppJWT_ParseRoundtrip(t *testing.T) {
	pemBytes := mustGenPEM(t)
	tok, err := BuildAppJWT("Iv23li-test", pemBytes)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.ParseWithClaims(tok, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Re-parse our own key
		key, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		if err != nil {
			return nil, err
		}
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims := parsed.Claims.(*jwt.RegisteredClaims)
	if claims.Issuer != "Iv23li-test" {
		t.Errorf("issuer = %q", claims.Issuer)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("missing iat/exp")
	}
	gap := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
	if gap < JWTLifetime-2*time.Second || gap > JWTLifetime+ClockSkew+2*time.Second {
		t.Errorf("exp-iat gap = %v, want ~%v", gap, JWTLifetime+ClockSkew)
	}
}

func TestBuildAppJWT_MissingIssuer(t *testing.T) {
	pemBytes := mustGenPEM(t)
	if _, err := BuildAppJWT("", pemBytes); err == nil {
		t.Error("expected error for empty issuer")
	}
}

func TestBuildAppJWT_BadPEM(t *testing.T) {
	if _, err := BuildAppJWT("x", []byte("not a pem")); err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestReadPrivateKey_RejectsBroadPerms(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "key.pem")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadPrivateKey(p)
	if err == nil {
		t.Fatal("expected perms error")
	}
	var pkErr *PrivateKeyPermsError
	if !errorAs(err, &pkErr) {
		t.Fatalf("expected *PrivateKeyPermsError, got %T: %v", err, err)
	}
	if !strings.Contains(pkErr.Error(), "chmod 600") {
		t.Errorf("error should suggest chmod: %v", pkErr)
	}
}

func TestReadPrivateKey_Accepts0600(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "key.pem")
	if err := os.WriteFile(p, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPrivateKey(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMintInstallationToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
			t.Errorf("api version header = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing bearer auth")
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Token{Token: "ghs_abc", ExpiresAt: time.Now().Add(time.Hour)})
	}))
	defer srv.Close()
	restore := setAPIBase(srv.URL)
	defer restore()

	tok, err := MintInstallationToken(context.Background(), "jwt", 12345, map[string]string{"contents": "write"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "ghs_abc" {
		t.Errorf("token = %q", tok.Token)
	}
}

func TestMintInstallationToken_4xxNoRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()
	restore := setAPIBase(srv.URL)
	defer restore()

	_, err := MintInstallationToken(context.Background(), "jwt", 12345, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestMintInstallationToken_5xxRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Token{Token: "ghs_after_retry", ExpiresAt: time.Now().Add(time.Hour)})
	}))
	defer srv.Close()
	restore := setAPIBase(srv.URL)
	defer restore()

	tok, err := MintInstallationToken(context.Background(), "jwt", 12345, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "ghs_after_retry" {
		t.Errorf("token = %q", tok.Token)
	}
	if got := atomic.LoadInt32(&calls); got < 3 {
		t.Errorf("expected at least 3 calls, got %d", got)
	}
}

func TestCacheRoundtrip(t *testing.T) {
	withTempHome(t, func() {
		e := &CacheEntry{
			Token:     "ghs_xyz",
			ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
			BotUserID: 42,
			BotLogin:  "my-app[bot]",
		}
		if err := WriteCache("my-app", e); err != nil {
			t.Fatal(err)
		}
		got, err := ReadCache("my-app")
		if err != nil {
			t.Fatal(err)
		}
		if got.Token != e.Token || got.BotUserID != e.BotUserID || !got.ExpiresAt.Equal(e.ExpiresAt) {
			t.Errorf("round-trip mismatch: got %+v want %+v", got, e)
		}
	})
}

func TestCacheFileMode0600(t *testing.T) {
	withTempHome(t, func() {
		e := &CacheEntry{Token: "x", ExpiresAt: time.Now().Add(time.Hour)}
		if err := WriteCache("my-app", e); err != nil {
			t.Fatal(err)
		}
		dir, _ := CacheDir()
		info, err := os.Stat(filepath.Join(dir, "my-app.json"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("file mode = %#o, want 0600", info.Mode().Perm())
		}
		dirInfo, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if dirInfo.Mode().Perm() != 0o700 {
			t.Errorf("dir mode = %#o, want 0700", dirInfo.Mode().Perm())
		}
	})
}

func TestCacheStale(t *testing.T) {
	fresh := &CacheEntry{ExpiresAt: time.Now().Add(time.Hour)}
	if fresh.Stale() {
		t.Error("1h-out entry should not be stale")
	}
	stale := &CacheEntry{ExpiresAt: time.Now().Add(time.Minute)}
	if !stale.Stale() {
		t.Error("1min-out entry should be stale")
	}
}

func TestResolveBotUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub accepts /users/<slug>[bot] with literal brackets; Go's
		// url.PathEscape leaves [ and ] unescaped because they're valid in
		// URL path segments per RFC 3986. Accept either form.
		path := r.URL.Path
		if !strings.HasSuffix(path, "/users/my-app[bot]") && !strings.HasSuffix(path, "/users/my-app%5Bbot%5D") {
			t.Errorf("unexpected path %q", path)
		}
		_ = json.NewEncoder(w).Encode(BotUser{ID: 42, Login: "my-app[bot]"})
	}))
	defer srv.Close()
	restore := setAPIBase(srv.URL)
	defer restore()

	bu, err := ResolveBotUser(context.Background(), "tok", "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if bu.ID != 42 {
		t.Errorf("id = %d", bu.ID)
	}
	if bu.Login != "my-app[bot]" {
		t.Errorf("login = %q", bu.Login)
	}
	want := "42+my-app[bot]@users.noreply.github.com"
	if got := BotCommitEmail(bu); got != want {
		t.Errorf("email = %q want %q", got, want)
	}
}

// ---- helpers ----

func setAPIBase(url string) func() {
	prev := APIBase
	APIBase = url
	return func() { APIBase = prev }
}

func withTempHome(t *testing.T, fn func()) {
	t.Helper()
	prevHome := os.Getenv("HOME")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Cleanup(func() { _ = os.Setenv("HOME", prevHome) })
	fn()
}

func errorAs(err error, target interface{}) bool {
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if assignTarget(err, target) {
			return true
		}
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return false
}

func assignTarget(err error, target interface{}) bool {
	// Tiny shim instead of importing reflect: we know the only target type in
	// this test is **PrivateKeyPermsError.
	if pk, ok := err.(*PrivateKeyPermsError); ok {
		if tgt, ok := target.(**PrivateKeyPermsError); ok {
			*tgt = pk
			return true
		}
	}
	_ = fmt.Sprint // keep import for any future debug prints
	return false
}
