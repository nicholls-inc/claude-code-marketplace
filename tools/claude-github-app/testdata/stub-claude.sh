#!/usr/bin/env bash
# Test stub: writes its env block to $CLAUDE_STUB_ENV_FILE (if set) and exits
# with $CLAUDE_STUB_EXIT (default 0). Used in launcher integration tests.

set -u

if [[ -n "${CLAUDE_STUB_ENV_FILE:-}" ]]; then
    # Capture all GH_*, GITHUB_*, GIT_* env vars (relevant subset)
    env | grep -E '^(GH_|GITHUB_|GIT_)' | sort > "$CLAUDE_STUB_ENV_FILE"
fi

# Capture argv if requested
if [[ -n "${CLAUDE_STUB_ARGV_FILE:-}" ]]; then
    printf '%s\n' "$@" > "$CLAUDE_STUB_ARGV_FILE"
fi

exit "${CLAUDE_STUB_EXIT:-0}"
