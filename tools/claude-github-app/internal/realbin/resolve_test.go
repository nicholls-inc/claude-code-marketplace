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

func TestResolveByName_FindsFirstOnPath(t *testing.T) {
	shimDir := t.TempDir()
	realDir := t.TempDir()
	makeExecutable(t, shimDir, "gh")
	realGh := makeExecutable(t, realDir, "gh")

	// shimDir first, realDir second. Pretend selfPath is shimDir/gh.
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+realDir)
	got, err := ResolveByName("gh", filepath.Join(shimDir, "gh"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(realGh)
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveByName_SkipsShimDirEvenIfFirst(t *testing.T) {
	shimDir := t.TempDir()
	realDir := t.TempDir()
	otherDir := t.TempDir()
	// shim, then a useless dir, then the real one
	shimGh := makeExecutable(t, shimDir, "gh")
	makeExecutable(t, realDir, "gh")
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+otherDir+string(os.PathListSeparator)+realDir)

	got, err := ResolveByName("gh", shimGh)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, shimDir) {
		t.Errorf("resolver returned shim binary: %q", got)
	}
}

func TestResolveByName_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	_, err := ResolveByName("definitely-not-a-real-binary-xyz", "")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestResolveByName_RejectsPathSeparator(t *testing.T) {
	_, err := ResolveByName("foo/bar", "")
	if err == nil || !strings.Contains(err.Error(), "bare filename") {
		t.Fatalf("expected bare-filename error, got %v", err)
	}
}

func TestResolveByName_EmptyPath(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := ResolveByName("gh", "")
	if err == nil || !strings.Contains(err.Error(), "PATH is empty") {
		t.Fatalf("expected PATH-empty error, got %v", err)
	}
}

func TestResolveByName_SkipsNonExecutable(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	// Non-executable in dir1
	if err := os.WriteFile(filepath.Join(dir1, "gh"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	realGh := makeExecutable(t, dir2, "gh")
	t.Setenv("PATH", dir1+string(os.PathListSeparator)+dir2)
	got, err := ResolveByName("gh", "")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(realGh)
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveByName_SkipsSymlinkToShim(t *testing.T) {
	shimDir := t.TempDir()
	aliasDir := t.TempDir()
	shimGh := makeExecutable(t, shimDir, "gh")
	// Create a symlink in aliasDir pointing to the shim
	if err := os.Symlink(shimGh, filepath.Join(aliasDir, "gh")); err != nil {
		t.Fatal(err)
	}
	realDir := t.TempDir()
	realGh := makeExecutable(t, realDir, "gh")

	t.Setenv("PATH", aliasDir+string(os.PathListSeparator)+realDir)
	got, err := ResolveByName("gh", shimGh)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(realGh)
	if got != want {
		t.Errorf("resolver followed symlink back to shim: got %q want %q", got, want)
	}
}
