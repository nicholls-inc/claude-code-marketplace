//go:build unix

package shim

import "syscall"

// syscallExec is the exec hook used by Run. It's a package variable so
// tests can swap it for an observer that captures (path, argv, env) without
// actually replacing the test process.
//
// Using exec (vs. fork+exec) preserves the original PID, stdio, signal
// disposition, and process group — critical because Claude Code's Bash
// tool relies on shared pgrp for SIGINT propagation.
var syscallExec = func(path string, argv []string, env []string) error {
	return syscall.Exec(path, argv, env)
}
