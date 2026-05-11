package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func stubPath(t *testing.T) string {
	t.Helper()
	// internal/launcher → ../../testdata/stub-claude.sh
	_, file, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(file)
	stub := filepath.Join(pkgDir, "..", "..", "testdata", "stub-claude.sh")
	abs, err := filepath.Abs(stub)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatalf("stub not found at %s: %v", abs, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		// Be self-healing on fresh checkouts where +x got dropped.
		_ = os.Chmod(abs, 0o755)
	}
	return abs
}

func TestRun_ExitCodePassthrough_Zero(t *testing.T) {
	code, err := Run(RunOpts{
		RealClaude: stubPath(t),
		Args:       []string{},
		Env:        append(os.Environ(), "CLAUDE_STUB_EXIT=0"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}

func TestRun_ExitCodePassthrough_NonZero(t *testing.T) {
	code, err := Run(RunOpts{
		RealClaude: stubPath(t),
		Args:       []string{},
		Env:        append(os.Environ(), "CLAUDE_STUB_EXIT=42"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 42 {
		t.Errorf("code = %d, want 42", code)
	}
}

func TestRun_EnvBlockReachesChild(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "env.out")
	code, err := Run(RunOpts{
		RealClaude: stubPath(t),
		Args:       []string{},
		Env: append(os.Environ(),
			"CLAUDE_STUB_EXIT=0",
			"CLAUDE_STUB_ENV_FILE="+envFile,
			"GH_TOKEN=ghs_test",
			"GITHUB_TOKEN=ghs_test",
			"GH_CONFIG_DIR=/tmp/fake-gh",
			"GIT_CONFIG_GLOBAL=/tmp/fake-git",
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
			"GH_PROMPT_DISABLED=1",
			"GH_NO_UPDATE_NOTIFIER=1",
		),
	})
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{
		"GH_CONFIG_DIR=/tmp/fake-gh",
		"GH_NO_UPDATE_NOTIFIER=1",
		"GH_PROMPT_DISABLED=1",
		"GH_TOKEN=ghs_test",
		"GITHUB_TOKEN=ghs_test",
		"GIT_CONFIG_GLOBAL=/tmp/fake-git",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
	sort.Strings(got)
	gotSet := map[string]bool{}
	for _, line := range got {
		gotSet[line] = true
	}
	for _, w := range want {
		if !gotSet[w] {
			t.Errorf("env block missing %q\nfull output:\n%s", w, string(data))
		}
	}
}

func TestRun_ArgvPassthrough(t *testing.T) {
	argvFile := filepath.Join(t.TempDir(), "argv.out")
	code, err := Run(RunOpts{
		RealClaude: stubPath(t),
		Args:       []string{"--flag", "value with spaces", "--bool"},
		Env: append(os.Environ(),
			"CLAUDE_STUB_EXIT=0",
			"CLAUDE_STUB_ARGV_FILE="+argvFile,
		),
	})
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	data, _ := os.ReadFile(argvFile)
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"--flag", "value with spaces", "--bool"}
	if len(got) != len(want) {
		t.Fatalf("argv count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("argv[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestSweepStaleTempDirs_RemovesOldDirs(t *testing.T) {
	tmp := os.TempDir()
	// Create a "stale" dir
	stale, err := os.MkdirTemp(tmp, "claude-github-app-*-test-stale")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(stale) })
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	// Create a "fresh" dir
	fresh, err := os.MkdirTemp(tmp, "claude-github-app-*-test-fresh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(fresh) })

	SweepStaleTempDirs()

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale dir not removed: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh dir wrongly removed: %v", err)
	}
}

func TestTempGitConfig_ContentsAndMode(t *testing.T) {
	path, cleanup, err := TempGitConfig(GitConfigOpts{
		Token:    "ghs_test",
		BotName:  "my-app[bot]",
		BotEmail: "42+my-app[bot]@users.noreply.github.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("git config mode = %#o, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	contents := string(data)
	for _, want := range []string{
		"name = my-app[bot]",
		"email = 42+my-app[bot]@users.noreply.github.com",
		"helper =",
		"extraHeader = Authorization: Bearer ghs_test",
	} {
		if !strings.Contains(contents, want) {
			t.Errorf("git config missing %q\ngot:\n%s", want, contents)
		}
	}
}

func TestTempGitConfig_NoBotIdentity(t *testing.T) {
	path, cleanup, err := TempGitConfig(GitConfigOpts{Token: "ghs_test"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "[user]") {
		t.Errorf("[user] block leaked despite no bot identity: %s", string(data))
	}
	if !strings.Contains(string(data), "Authorization: Bearer ghs_test") {
		t.Errorf("auth header missing")
	}
}

func TestRenderGitConfig_EmptyTokenRefused(t *testing.T) {
	if _, err := RenderGitConfig(GitConfigOpts{}); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestRenderGitConfig_OmitsUserBlockWhenIdentityMissing(t *testing.T) {
	s, err := RenderGitConfig(GitConfigOpts{Token: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s, "[user]") {
		t.Errorf("expected no [user] block, got:\n%s", s)
	}
	if !strings.Contains(s, "Authorization: Bearer x") {
		t.Errorf("auth header missing:\n%s", s)
	}
}

func TestWriteGitConfigAtomic_CreatesFileWithModeAndContents(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "sub", "gitconfig")
	err := WriteGitConfigAtomic(dst, GitConfigOpts{
		Token:    "ghs_abc",
		BotName:  "my-app[bot]",
		BotEmail: "1+my-app[bot]@users.noreply.github.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %#o, want 0600", info.Mode().Perm())
	}
	parent, err := os.Stat(filepath.Dir(dst))
	if err != nil {
		t.Fatal(err)
	}
	if parent.Mode().Perm() != 0o700 {
		t.Errorf("parent dir mode = %#o, want 0700", parent.Mode().Perm())
	}
	data, _ := os.ReadFile(dst)
	contents := string(data)
	if !strings.Contains(contents, "Authorization: Bearer ghs_abc") {
		t.Errorf("auth header missing:\n%s", contents)
	}
	if !strings.Contains(contents, "name = my-app[bot]") {
		t.Errorf("user.name missing:\n%s", contents)
	}
}

func TestWriteGitConfigAtomic_OverwritesExistingFile(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "gitconfig")
	if err := WriteGitConfigAtomic(dst, GitConfigOpts{Token: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteGitConfigAtomic(dst, GitConfigOpts{Token: "second"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	if strings.Contains(string(data), "first") {
		t.Errorf("old token leaked:\n%s", string(data))
	}
	if !strings.Contains(string(data), "second") {
		t.Errorf("new token missing:\n%s", string(data))
	}
}

func TestWriteGitConfigAtomic_NoTempFileLeak(t *testing.T) {
	// After a successful write, the parent dir should contain ONLY the
	// destination file — no .gitconfig-* temp residue.
	dir := t.TempDir()
	dst := filepath.Join(dir, "gitconfig")
	if err := WriteGitConfigAtomic(dst, GitConfigOpts{Token: "x"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "gitconfig" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected only [gitconfig], got %v", names)
	}
}

func TestTempGHConfigDir_ModeAndCleanup(t *testing.T) {
	dir, cleanup, err := TempGHConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("gh config dir mode = %#o, want 0700", info.Mode().Perm())
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove dir: %v", err)
	}
}
