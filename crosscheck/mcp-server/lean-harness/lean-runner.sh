#!/usr/bin/env bash
# Runtime entrypoint for the Lean harness Docker image.
#
# Usage (invoked by the MCP server inside the container):
#   lean-runner.sh check /work/program.lean   # parse + typecheck via lake build
#   lean-runner.sh run   /work/program.lean   # build then `lake env lean --run`
#   lean-runner.sh test  /work/program.lean   # build then `lake test`
#
# Each subcommand swaps the user file into the harness module so Mathlib's
# pre-warmed oleans are reused, then runs the relevant lake invocation.

set -euo pipefail

if [[ "$#" -lt 2 ]]; then
  echo "lean-runner.sh: expected 2 args (subcommand + /work/program.lean)" >&2
  echo "got: $*" >&2
  exit 2
fi

subcommand="$1"
program="$2"

if [[ ! -f "$program" ]]; then
  echo "lean-runner.sh: program file not found: $program" >&2
  exit 2
fi

# Swap the user's source into the harness module slot.
cp "$program" /harness/Crosscheck/Program.lean
cd /harness

case "$subcommand" in
  check)
    # `lake build` performs parse + elaboration + typecheck on every
    # transitively-imported module. With Mathlib already cached, this is the
    # fastest way to surface the user's compile-time errors.
    exec lake build Crosscheck
    ;;
  run)
    # Build first to surface compile errors before execution; then run the
    # user's `main`. Failures during build short-circuit before run.
    lake build Crosscheck
    exec lake env lean --run /harness/Crosscheck/Program.lean
    ;;
  test)
    # 3b-α: aliased to `lake build`. `lake test` requires a `test` driver in
    # lakefile.lean that has not yet been wired. The MCP `lean_test` tool
    # description carries the same caveat. Sub-phase 3b-β adds a #guard-driven
    # test target and switches both this branch and the tool to true `lake
    # test` semantics. Today, `#guard` failures still surface (as build
    # errors) — silent downgrade only affects test-runner semantics like
    # per-case reporting and selective filtering.
    exec lake build Crosscheck
    ;;
  *)
    echo "lean-runner.sh: unknown subcommand '$subcommand' (expected: check|run|test)" >&2
    exit 2
    ;;
esac
