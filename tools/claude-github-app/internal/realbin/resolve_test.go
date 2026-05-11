package realbin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeExecutable writes a no-op script and chmods +x, returning its path.
func makeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolve_OverrideUsesPathVerbatim(t *testing.T) {
	tmp := t.TempDir()
	bin := makeExecutable(t, tmp, "claude-real")
	got, err := Resolve(bin, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, _ := filepath.EvalSymlinks(bin)
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolve_FollowsSymlink(t *testing.T) {
	tmp := t.TempDir()
	realBin := makeExecutable(t, tmp, "claude-real")
	link := filepath.Join(tmp, "claude-link")
	if err := os.Symlink(realBin, link); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(link, "")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(realBin)
	if got != want {
		t.Errorf("symlink not followed: got %q want %q", got, want)
	}
}

func TestResolve_SelfLoopDetected(t *testing.T) {
	tmp := t.TempDir()
	bin := makeExecutable(t, tmp, "claude")
	_, err := Resolve(bin, bin)
	if err == nil || !strings.Contains(err.Error(), "self-loop") {
		t.Fatalf("expected self-loop error, got %v", err)
	}
}

func TestResolve_SelfLoopGuardSoftFailsOnEmptySelf(t *testing.T) {
	tmp := t.TempDir()
	bin := makeExecutable(t, tmp, "claude")
	if _, err := Resolve(bin, ""); err != nil {
		t.Fatalf("empty selfPath should skip guard, got %v", err)
	}
}

func TestResolve_RejectsDirectory(t *testing.T) {
	tmp := t.TempDir()
	_, err := Resolve(tmp, "")
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func TestResolve_RejectsNonExecutable(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "data")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve(p, "")
	if err == nil || !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("expected non-executable error, got %v", err)
	}
}

func TestResolve_MissingFile(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "absent"), "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
