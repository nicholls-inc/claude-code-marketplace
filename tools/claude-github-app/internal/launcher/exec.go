package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// SweepMaxAge bounds how long a stranded temp dir may survive before the next
// wrapper launch reaps it. Enforces Properties §D.
const SweepMaxAge = 24 * time.Hour

// SweepStaleTempDirs removes claude-github-app-* temp dirs in os.TempDir()
// older than SweepMaxAge. Best-effort; errors are silently swallowed so a
// stranded permission-denied entry can't break a fresh launch.
func SweepStaleTempDirs() {
	tmp := os.TempDir()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-SweepMaxAge)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "claude-github-app-") {
			continue
		}
		full := filepath.Join(tmp, e.Name())
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(full)
		}
	}
}

// RunOpts controls a single exec.
type RunOpts struct {
	RealClaude string   // resolved absolute path
	Args       []string // argv[1:] for the child
	Env        []string // full env for the child (typically os.Environ() + injections)
}

// Run starts the real claude binary, forwards signals, waits, and returns the
// exit code per Unix convention (128+signum on signal death).
//
// Caller is responsible for cleanups (defer cleanup() before calling Run).
// SIGPIPE is unconditionally ignored by this process; the child inherits the
// default disposition so it can detect broken pipes if it wants to.
func Run(opts RunOpts) (int, error) {
	// Ignore SIGPIPE in the wrapper so a status-line write to a closed stderr
	// during a `claude --version | head -1` style invocation doesn't kill us.
	signal.Ignore(syscall.SIGPIPE)

	cmd := exec.Command(opts.RealClaude, opts.Args...)
	cmd.Env = opts.Env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Deliberately do NOT set SysProcAttr.Setpgid: shared pgrp is how the
	// kernel delivers SIGWINCH to the TUI on terminal resize.

	if err := cmd.Start(); err != nil {
		return 127, fmt.Errorf("start claude (%s): %w", opts.RealClaude, err)
	}

	// Forward signals to the child. SIGTSTP is special: we forward it then
	// raise SIGSTOP on ourselves so shell job control sees the wrapper as
	// stopped too; on SIGCONT we forward and resume implicitly.
	sigs := make(chan os.Signal, 8)
	signal.Notify(sigs,
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT,
		syscall.SIGTSTP, syscall.SIGCONT,
	)
	done := make(chan struct{})
	go forwardSignals(cmd.Process, sigs, done)

	waitErr := cmd.Wait()
	close(done)
	signal.Stop(sigs)

	if waitErr == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				return 128 + int(ws.Signal()), nil
			}
			return ws.ExitStatus(), nil
		}
		return exitErr.ExitCode(), nil
	}
	return 1, waitErr
}

func forwardSignals(child *os.Process, sigs <-chan os.Signal, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case s, ok := <-sigs:
			if !ok {
				return
			}
			switch s {
			case syscall.SIGTSTP:
				_ = child.Signal(syscall.SIGTSTP)
				// Stop ourselves so shell job control sees both parent and
				// child as stopped. Resume happens automatically on SIGCONT.
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
			default:
				_ = child.Signal(s)
			}
		}
	}
}
