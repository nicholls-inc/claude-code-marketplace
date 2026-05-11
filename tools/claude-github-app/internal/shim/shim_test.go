package shim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/tools/claude-github-app/internal/token"
)

func TestApplyInjections_OverridesAndAppends(t *testing.T) {
	env := []string{"FOO=1", "BAR=2", "PATH=/usr/bin"}
	got := applyInjections(env, map[string]string{
		"BAR":      "OVERRIDDEN",
		"NEW_KEY":  "x",
		"NEW_KEY2": "y",
	})
	sort.Strings(got)
	want := []string{"BAR=OVERRIDDEN", "FOO=1", "NEW_KEY2=y", "NEW_KEY=x", "PATH=/usr/bin"}
	if !equalStringSlices(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestApplyInjections_NilMapPassthrough(t *testing.T) {
	env := []string{"FOO=1"}
	got := applyInjections(env, nil)
	if len(got) != 1 || got[0] != "FOO=1" {
		t.Errorf("nil map should be identity, got %v", got)
	}
}

func TestApplyInjections_HandlesMalformedEnvEntry(t *testing.T) {
	// Entries without '=' (rare but legal-ish) should pass through.
	env := []string{"NOEQUALS", "VALID=1"}
	got := applyInjections(env, map[string]string{"VALID": "2"})
	sort.Strings(got)
	want := []string{"NOEQUALS", "VALID=2"}
	if !equalStringSlices(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

// TestRun_PassthroughWhenNoConfig — when no config file exists, Run should
// resolve the real binary and exec it unmodified with the parent env.
func TestRun_PassthroughWhenNoConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim is unix-only")
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// No config file at HOME/.config/claude-github-app/config.toml

	realDir := t.TempDir()
	realBin := makeStub(t, realDir, "gh-real")
	// Pretend our shim is in a separate dir and we have nothing else
	// on PATH besides realDir.
	t.Setenv("PATH", realDir)

	var capturedPath string
	var capturedArgv []string
	var capturedEnv []string
	syscallExec = func(path string, argv []string, env []string) error {
		capturedPath = path
		capturedArgv = argv
		capturedEnv = env
		return nil
	}
	t.Cleanup(restoreSyscallExec)

	// Use a binary name that exists in realDir
	code := Run(Options{
		RealName: "gh-real",
		Args:     []string{"--version"},
		SelfPath: "", // skip self-loop guard
		Inject: func(_ *token.CacheEntry, _ *config.App) (map[string]string, func(), error) {
			t.Fatal("Inject should not be called when no app maps to CWD")
			return nil, nil, nil
		},
	})
	if code != 0 {
		t.Fatalf("code=%d, want 0", code)
	}
	wantReal, _ := filepath.EvalSymlinks(realBin)
	if capturedPath != wantReal {
		t.Errorf("exec path = %q, want %q", capturedPath, wantReal)
	}
	if !equalStringSlices(capturedArgv, []string{wantReal, "--version"}) {
		t.Errorf("argv = %v", capturedArgv)
	}
	// Env must be parent env, no token injection.
	for _, e := range capturedEnv {
		if strings.HasPrefix(e, "GH_TOKEN=") || strings.HasPrefix(e, "GITHUB_TOKEN=") {
			t.Errorf("unexpected token leak in passthrough: %q", e)
		}
	}
}

// TestRun_InjectionWhenAppMapped — when config maps CWD to an app and the
// token cache is fresh, Run should call Inject with the cached entry and
// pass returned env into the exec call.
func TestRun_InjectionWhenAppMapped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim is unix-only")
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Write a config that maps t.TempDir() (which we'll chdir into) to an app.
	mappedDir := t.TempDir()
	if err := os.Chdir(mappedDir); err != nil {
		t.Fatal(err)
	}
	// Restore CWD after the test so subsequent tests still find their paths.
	origCwd, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(origCwd)
	})

	// Write a private key (PEM) — even a dummy that won't parse if we
	// reach mint. We will NOT reach mint because the cache is fresh.
	keyPath := filepath.Join(tmpHome, "fake.pem")
	if err := os.WriteFile(keyPath, []byte("-----BEGIN DUMMY-----\n-----END DUMMY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfgDir := filepath.Join(tmpHome, ".config", "claude-github-app")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfgBody := `
[[apps]]
name = "my-app"
client_id = "Iv23test"
installation_id = 1234
private_key_file = "` + keyPath + `"

[[mappings]]
path = "` + mappedDir + `"
app = "my-app"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(cfgBody), 0o600); err != nil {
		t.Fatal(err)
	}

	// Seed a fresh token cache so EnsureToken doesn't try to mint.
	cacheDir := filepath.Join(tmpHome, ".cache", "claude-github-app")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	entry := token.CacheEntry{
		Token:     "ghs_cached_token",
		ExpiresAt: time.Now().Add(30 * time.Minute),
		BotUserID: 99,
		BotLogin:  "my-app[bot]",
	}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(filepath.Join(cacheDir, "my-app.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	realDir := t.TempDir()
	realBin := makeStub(t, realDir, "gh-real")
	t.Setenv("PATH", realDir)

	var capturedEnv []string
	var capturedPath string
	syscallExec = func(path string, argv []string, env []string) error {
		capturedPath = path
		capturedEnv = env
		return nil
	}
	t.Cleanup(restoreSyscallExec)

	var injectedEntryToken string
	code := Run(Options{
		RealName: "gh-real",
		Args:     []string{},
		SelfPath: "",
		Inject: func(e *token.CacheEntry, _ *config.App) (map[string]string, func(), error) {
			injectedEntryToken = e.Token
			return map[string]string{
				"GH_TOKEN":     e.Token,
				"GITHUB_TOKEN": e.Token,
			}, nil, nil
		},
	})
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if injectedEntryToken != "ghs_cached_token" {
		t.Errorf("Inject got token %q, want cached", injectedEntryToken)
	}
	wantReal, _ := filepath.EvalSymlinks(realBin)
	if capturedPath != wantReal {
		t.Errorf("exec path = %q, want %q", capturedPath, wantReal)
	}
	foundGH, foundGitHub := false, false
	for _, e := range capturedEnv {
		if e == "GH_TOKEN=ghs_cached_token" {
			foundGH = true
		}
		if e == "GITHUB_TOKEN=ghs_cached_token" {
			foundGitHub = true
		}
	}
	if !foundGH || !foundGitHub {
		t.Errorf("env missing token injection: %v", capturedEnv)
	}
}

// TestRun_CleanupCalledBeforeExec — Inject's returned cleanup must run
// before exec replaces the process (since exec discards our stack).
func TestRun_CleanupCalledBeforeExec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim is unix-only")
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	mappedDir := t.TempDir()
	if err := os.Chdir(mappedDir); err != nil {
		t.Fatal(err)
	}
	origCwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	keyPath := filepath.Join(tmpHome, "fake.pem")
	_ = os.WriteFile(keyPath, []byte("dummy"), 0o600)

	cfgDir := filepath.Join(tmpHome, ".config", "claude-github-app")
	_ = os.MkdirAll(cfgDir, 0o700)
	cfgBody := `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "` + keyPath + `"

[[mappings]]
path = "` + mappedDir + `"
app = "a"
`
	_ = os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(cfgBody), 0o600)

	cacheDir := filepath.Join(tmpHome, ".cache", "claude-github-app")
	_ = os.MkdirAll(cacheDir, 0o700)
	entry := token.CacheEntry{Token: "t", ExpiresAt: time.Now().Add(time.Hour)}
	d, _ := json.Marshal(entry)
	_ = os.WriteFile(filepath.Join(cacheDir, "a.json"), d, 0o600)

	realDir := t.TempDir()
	makeStub(t, realDir, "fake")
	t.Setenv("PATH", realDir)

	var sequence []string
	syscallExec = func(_ string, _ []string, _ []string) error {
		sequence = append(sequence, "exec")
		return nil
	}
	t.Cleanup(restoreSyscallExec)

	Run(Options{
		RealName: "fake",
		Args:     nil,
		SelfPath: "",
		Inject: func(_ *token.CacheEntry, _ *config.App) (map[string]string, func(), error) {
			return nil, func() {
				sequence = append(sequence, "cleanup")
			}, nil
		},
	})
	if !equalStringSlices(sequence, []string{"cleanup", "exec"}) {
		t.Errorf("expected cleanup before exec, got %v", sequence)
	}
}

// ---- helpers ----

func makeStub(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func restoreSyscallExec() {
	// Reinitialise to the default exec hook. We can't reference syscall.Exec
	// here cross-platform; instead, set it to a no-op so any leak doesn't
	// nuke a parallel test process. Tests that need exec set their own hook.
	syscallExec = func(_ string, _ []string, _ []string) error { return nil }
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
