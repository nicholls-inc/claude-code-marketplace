package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

func TestInjectGitAuth_WritesPerAppConfigAndReturnsEnv(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	entry := &token.CacheEntry{
		Token:     "ghs_abc",
		ExpiresAt: time.Now().Add(time.Hour),
		BotUserID: 42,
		BotLogin:  "my-app[bot]",
	}
	app := &config.App{Name: "my-app"}

	env, cleanup, err := injectGitAuth(entry, app)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup != nil {
		t.Errorf("git shim should not request a cleanup (gitconfig is persistent)")
	}

	wantPath := filepath.Join(tmpHome, ".cache", "claude-github-app", "my-app-gitconfig")
	if env["GIT_CONFIG_GLOBAL"] != wantPath {
		t.Errorf("GIT_CONFIG_GLOBAL = %q, want %q", env["GIT_CONFIG_GLOBAL"], wantPath)
	}
	if env["GIT_CONFIG_NOSYSTEM"] != "1" {
		t.Errorf("GIT_CONFIG_NOSYSTEM should be 1, got %q", env["GIT_CONFIG_NOSYSTEM"])
	}

	info, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("gitconfig not written: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("gitconfig mode = %#o, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(wantPath)
	contents := string(data)
	for _, want := range []string{
		"Authorization: Bearer ghs_abc",
		"name = my-app[bot]",
		"email = 42+my-app[bot]@users.noreply.github.com",
	} {
		if !strings.Contains(contents, want) {
			t.Errorf("gitconfig missing %q:\n%s", want, contents)
		}
	}
}

func TestInjectGitAuth_NoBotIdentitySkipsUserBlock(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	entry := &token.CacheEntry{
		Token:     "ghs_x",
		ExpiresAt: time.Now().Add(time.Hour),
		// no BotUserID/BotLogin
	}
	app := &config.App{Name: "anon-app"}

	if _, _, err := injectGitAuth(entry, app); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(tmpHome, ".cache", "claude-github-app", "anon-app-gitconfig")
	data, _ := os.ReadFile(wantPath)
	if strings.Contains(string(data), "[user]") {
		t.Errorf("[user] block should not be written without bot identity:\n%s", string(data))
	}
}

func TestInjectGitAuth_RewritesOnSecondCall(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	app := &config.App{Name: "r-app"}

	first := &token.CacheEntry{Token: "first", ExpiresAt: time.Now().Add(time.Hour)}
	second := &token.CacheEntry{Token: "second", ExpiresAt: time.Now().Add(time.Hour)}

	if _, _, err := injectGitAuth(first, app); err != nil {
		t.Fatal(err)
	}
	if _, _, err := injectGitAuth(second, app); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(tmpHome, ".cache", "claude-github-app", "r-app-gitconfig")
	data, _ := os.ReadFile(wantPath)
	if strings.Contains(string(data), "Bearer first") {
		t.Errorf("stale token still present after refresh:\n%s", string(data))
	}
	if !strings.Contains(string(data), "Bearer second") {
		t.Errorf("new token not written:\n%s", string(data))
	}
}
