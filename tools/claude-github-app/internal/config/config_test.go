package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_MinimalValid(t *testing.T) {
	p := writeConfig(t, `
[[apps]]
name = "my-app"
client_id = "Iv23li-test"
installation_id = 12345
private_key_file = "/tmp/key.pem"

[[mappings]]
path = "/Users/h/repos/foo"
app = "my-app"
`)
	c, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Apps) != 1 || c.Apps[0].Name != "my-app" {
		t.Fatalf("apps: %+v", c.Apps)
	}
	if got := c.Apps[0].Permissions["contents"]; got != "write" {
		t.Errorf("default permissions not applied; contents=%q", got)
	}
	if len(c.Apps[0].Permissions) != 4 {
		t.Errorf("default permissions count = %d, want 4", len(c.Apps[0].Permissions))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected IsNotExist, got %v", err)
	}
}

func TestLoad_DuplicateMappingPaths(t *testing.T) {
	p := writeConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/x"
app = "a"

[[mappings]]
path = "/x/"
app = "a"
`)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("expected duplicate path error, got %v", err)
	}
}

func TestLoad_MappingReferencesUnknownApp(t *testing.T) {
	p := writeConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/x"
app = "does-not-exist"
`)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("expected unknown-app error, got %v", err)
	}
}

func TestLoad_DefaultAppMustExist(t *testing.T) {
	p := writeConfig(t, `
default_app = "ghost"

[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"
`)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "default_app") {
		t.Fatalf("expected default_app error, got %v", err)
	}
}

func TestLoad_ExplicitPermissionsRespected(t *testing.T) {
	p := writeConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[apps.permissions]
contents = "read"
`)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Apps[0].Permissions["contents"] != "read" {
		t.Errorf("explicit permission overridden: %v", c.Apps[0].Permissions)
	}
	if _, ok := c.Apps[0].Permissions["pull_requests"]; ok {
		t.Errorf("default permissions leaked into explicit map")
	}
}

func TestMatch_LongestPrefixWins(t *testing.T) {
	c := mustConfig(t, `
[[apps]]
name = "outer"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[apps]]
name = "inner"
client_id = "y"
installation_id = 2
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/Users/h/repos"
app = "outer"

[[mappings]]
path = "/Users/h/repos/specific"
app = "inner"
`)
	got := c.Match("/Users/h/repos/specific/subdir")
	if got == nil || got.Name != "inner" {
		t.Errorf("longest-prefix mismatch: got %v", got)
	}
	got = c.Match("/Users/h/repos/other")
	if got == nil || got.Name != "outer" {
		t.Errorf("outer should win: got %v", got)
	}
}

func TestMatch_NoFalsePrefixOnSimilarNames(t *testing.T) {
	c := mustConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/Users/h/foo"
app = "a"
`)
	if got := c.Match("/Users/h/foo-bar"); got != nil {
		t.Errorf("matched on partial path component: %v", got)
	}
	if got := c.Match("/Users/h/foo"); got == nil {
		t.Errorf("exact match failed")
	}
	if got := c.Match("/Users/h/foo/x"); got == nil {
		t.Errorf("child match failed")
	}
}

func TestMatch_TrailingSlashInConfig(t *testing.T) {
	c := mustConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/Users/h/foo/"
app = "a"
`)
	if got := c.Match("/Users/h/foo/bar"); got == nil {
		t.Errorf("trailing slash in config path should not break matching")
	}
}

func TestMatch_DefaultAppFallback(t *testing.T) {
	c := mustConfig(t, `
default_app = "fallback"

[[apps]]
name = "fallback"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[apps]]
name = "specific"
client_id = "y"
installation_id = 2
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/Users/h/foo"
app = "specific"
`)
	got := c.Match("/tmp")
	if got == nil || got.Name != "fallback" {
		t.Errorf("default app fallback failed: %v", got)
	}
}

func TestMatch_NoMatchNoDefault(t *testing.T) {
	c := mustConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"

[[mappings]]
path = "/Users/h/foo"
app = "a"
`)
	if got := c.Match("/tmp"); got != nil {
		t.Errorf("expected no match, got %v", got)
	}
}

func TestMatch_SymlinkFallback(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	link := filepath.Join(tmp, "link")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, `
[[apps]]
name = "a"
client_id = "x"
installation_id = 1
private_key_file = "/tmp/k.pem"
`)
	// Inject a mapping pointing at the real path. Re-run normalization so
	// pathResolved is populated (covers the macOS /var → /private/var alias).
	c.Mappings = []Mapping{{Path: real, App: "a"}}
	if err := c.normalize(); err != nil {
		t.Fatal(err)
	}

	// CWD is the symlink; logical match fails, EvalSymlinks fallback succeeds.
	if got := c.Match(link); got == nil || got.Name != "a" {
		t.Errorf("EvalSymlinks fallback failed: got %v", got)
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no HOME")
	}
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"/abs/path", "/abs/path"},
		{"relative", "relative"},
	}
	for _, c := range cases {
		got, err := expandTilde(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("%q: got %q want %q", c.in, got, c.want)
		}
	}
}

func mustConfig(t *testing.T, body string) *Config {
	t.Helper()
	p := writeConfig(t, body)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
