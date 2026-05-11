#!/usr/bin/env bash
# /journal-context — deterministic walk of JOURNAL.md from a path to repo root.
# Invariants pinned in:
#   crosscheck/skills/journal-context/docs/invariants/journal-context.md
#
# Usage: walk.sh [path]
#   path defaults to current working directory.
set -uo pipefail

path="${1:-.}"

if [ ! -e "$path" ]; then
  printf 'journal-context: path not found: %s\n' "$path" >&2
  exit 2
fi

# Resolve starting directory: dir itself if a dir, parent dir if a file.
if [ -d "$path" ]; then
  start="$path"
else
  start="$(dirname -- "$path")"
fi

# Find the enclosing git toplevel in canonical form. If the path is not
# inside any git repository, emit the I6 not-in-repo message and stop.
if ! canon_top="$(git -C "$start" rev-parse --show-toplevel 2>/dev/null)"; then
  printf '# no JOURNAL.md found: %s is not inside a git repository\n' "$path"
  exit 0
fi

# Walk uses logical parent-of (preserves user-supplied symlinks in the path).
# Termination compares canonical forms so the walk stops at the git toplevel
# even when the user invoked through a symlinked path or worktree.
logical_start="$(cd -- "$start" && pwd -L)"

emitted=0
dir="$logical_start"

while :; do
  jpath="$dir/JOURNAL.md"
  if [ -f "$jpath" ]; then
    # Path relative to repo root for the delimiter (canonical form).
    if canon_dir="$(cd -- "$dir" 2>/dev/null && pwd -P)"; then
      rel="${canon_dir#"$canon_top"}"
      rel="${rel#/}"
    else
      rel=""
    fi
    if [ -n "$rel" ]; then
      printf '=== %s/JOURNAL.md ===\n' "$rel"
    else
      printf '=== JOURNAL.md ===\n'
    fi
    cat -- "$jpath"
    # Guarantee newline separation between this file and the next delimiter
    # (or end of output). At most one blank line per file; visually clean.
    printf '\n'
    emitted=$((emitted + 1))
  fi

  # Termination: have we processed the toplevel directory? Compare canonical
  # forms so the logical walk terminates correctly through symlinks.
  if canon_dir="$(cd -- "$dir" 2>/dev/null && pwd -P)"; then
    if [ "$canon_dir" = "$canon_top" ]; then
      break
    fi
  else
    # Can't reach this directory — defensive break rather than loop forever.
    break
  fi

  parent="$(dirname -- "$dir")"
  if [ "$parent" = "$dir" ]; then
    # Walked above / without hitting toplevel — should not happen for paths
    # inside a git repo, but break defensively.
    break
  fi
  dir="$parent"
done

if [ "$emitted" -eq 0 ]; then
  printf '# no JOURNAL.md found above %s\n' "$logical_start"
fi
