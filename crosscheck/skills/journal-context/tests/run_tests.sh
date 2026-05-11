#!/usr/bin/env bash
# Tests for /journal-context, anchored to docs/invariants/journal-context.md
# Each section is tagged with `# Invariant Ix: <Name>` so a future
# bidirectional coverage gate can map tests <-> invariants by string match.
set -uo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SKILL_DIR="$(cd -- "$HERE/.." && pwd -P)"
WALK="$SKILL_DIR/scripts/walk.sh"

if [ ! -f "$WALK" ]; then
  printf 'cannot find walk.sh at %s\n' "$WALK" >&2
  exit 2
fi

TMP_BASE="$(mktemp -d "${TMPDIR:-/tmp}/journal-context-tests.XXXXXX")"
trap 'rm -rf -- "$TMP_BASE"' EXIT

pass=0
fail=0

assert_eq() {
  local label="$1" actual="$2" expected="$3"
  if [ "$actual" = "$expected" ]; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    expected: %q\n' "$expected"
    printf '    actual:   %q\n' "$actual"
    fail=$((fail + 1))
  fi
}

assert_contains() {
  local label="$1" haystack="$2" needle="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    expected to contain: %q\n' "$needle"
    printf '    actual:               %q\n' "$haystack"
    fail=$((fail + 1))
  fi
}

assert_not_contains() {
  local label="$1" haystack="$2" needle="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    expected NOT to contain: %q\n' "$needle"
    fail=$((fail + 1))
  fi
}

build_deep_fixture() {
  local root="$1"
  mkdir -p "$root/a/b/c"
  git -C "$root" init -q -b main 2>/dev/null || git -C "$root" init -q
  printf 'root journal\n' > "$root/JOURNAL.md"
  printf 'level a journal\n' > "$root/a/JOURNAL.md"
  printf 'level b journal\n' > "$root/a/b/JOURNAL.md"
  printf 'a deep file\n' > "$root/a/b/c/file.txt"
}

# -----------------------------------------------------------------------------
# Test 1 — walk shape + ordering
# Invariant I1: walk covers path-dir up to git toplevel inclusive
# Invariant I2: emit order is deepest-shard first
# -----------------------------------------------------------------------------
printf 'Test 1 (I1 walk shape, I2 ordering):\n'
T1="$TMP_BASE/t1"
build_deep_fixture "$T1"
out_t1="$(bash "$WALK" "$T1/a/b/c/file.txt")"
assert_contains "I1: output includes root JOURNAL.md content" "$out_t1" "root journal"
assert_contains "I1: output includes level-a JOURNAL.md content" "$out_t1" "level a journal"
assert_contains "I1: output includes level-b JOURNAL.md content" "$out_t1" "level b journal"

order_test="$(printf '%s\n' "$out_t1" | awk '
  /level b journal/   { o["b"]=NR }
  /level a journal/   { o["a"]=NR }
  /root journal/      { o["r"]=NR }
  END {
    if (o["b"] && o["a"] && o["r"] && o["b"] < o["a"] && o["a"] < o["r"])
      print "ok"
    else
      print "bad: b=" o["b"] " a=" o["a"] " r=" o["r"]
  }
')"
assert_eq "I2: deepest-first ordering (b < a < root)" "$order_test" "ok"

# -----------------------------------------------------------------------------
# Test 2 — determinism
# Invariant I3: same input + tree → byte-identical output
# -----------------------------------------------------------------------------
printf 'Test 2 (I3 determinism):\n'
T2="$TMP_BASE/t2"
build_deep_fixture "$T2"
out_a="$(bash "$WALK" "$T2/a/b/c/file.txt")"
out_b="$(bash "$WALK" "$T2/a/b/c/file.txt")"
assert_eq "I3: two runs produce byte-identical output" "$out_a" "$out_b"

# -----------------------------------------------------------------------------
# Test 3 — read-only
# Invariant I4: no filesystem or git mutation
# -----------------------------------------------------------------------------
printf 'Test 3 (I4 read-only):\n'
T3="$TMP_BASE/t3"
build_deep_fixture "$T3"
hash_before="$(find "$T3" -type f -print0 2>/dev/null | xargs -0 shasum 2>/dev/null | sort)"
status_before="$(git -C "$T3" status --porcelain)"
bash "$WALK" "$T3/a/b/c/file.txt" >/dev/null
hash_after="$(find "$T3" -type f -print0 2>/dev/null | xargs -0 shasum 2>/dev/null | sort)"
status_after="$(git -C "$T3" status --porcelain)"
assert_eq "I4: file content hashes unchanged after walk" "$hash_before" "$hash_after"
assert_eq "I4: git status unchanged after walk" "$status_before" "$status_after"

# -----------------------------------------------------------------------------
# Test 4 — symlink handling
# Invariant I5: symlinks do not redirect the walk
# -----------------------------------------------------------------------------
printf 'Test 4 (I5 symlinks):\n'
T4="$TMP_BASE/t4"
build_deep_fixture "$T4"
ln -s "$T4" "$TMP_BASE/symlinked_t4"
out_t4="$(bash "$WALK" "$TMP_BASE/symlinked_t4/a/b/c/file.txt")"
assert_contains "I5: walk-through-symlink reaches root journal" "$out_t4" "root journal"
assert_contains "I5: walk-through-symlink reaches level-a" "$out_t4" "level a journal"
assert_contains "I5: walk-through-symlink reaches level-b" "$out_t4" "level b journal"
delim_count="$(printf '%s\n' "$out_t4" | grep -c '^=== ' || true)"
assert_eq "I5: terminates with exactly 3 file delimiters" "$delim_count" "3"

# -----------------------------------------------------------------------------
# Test 5a — empty case: not in a git repo
# Invariant I6: zero-journal walk emits an explicit message
# -----------------------------------------------------------------------------
printf 'Test 5a (I6 not in a git repo):\n'
T5A="$TMP_BASE/t5a"
mkdir -p "$T5A/inner"
out_t5a="$(bash "$WALK" "$T5A/inner")"
assert_contains "I6: not-in-repo message is explicit" "$out_t5a" "not inside a git repository"

# -----------------------------------------------------------------------------
# Test 5b — empty case: inside a git repo, no journals
# Invariant I6: zero-journal walk emits an explicit message
# -----------------------------------------------------------------------------
printf 'Test 5b (I6 in repo, no journals):\n'
T5B="$TMP_BASE/t5b"
mkdir -p "$T5B/inner"
git -C "$T5B" init -q
out_t5b="$(bash "$WALK" "$T5B/inner")"
assert_contains "I6: no-journal-found message is explicit" "$out_t5b" "no JOURNAL.md found above"

# -----------------------------------------------------------------------------
# Test 6 — delimiter shape
# Invariant I7: each file's content is preceded by an === <path> === delimiter
# -----------------------------------------------------------------------------
printf 'Test 6 (I7 delimiter shape):\n'
T6="$TMP_BASE/t6"
build_deep_fixture "$T6"
out_t6="$(bash "$WALK" "$T6/a/b/c/file.txt")"
assert_contains "I7: root delimiter '=== JOURNAL.md ==='" "$out_t6" "=== JOURNAL.md ==="
assert_contains "I7: level-a delimiter '=== a/JOURNAL.md ==='" "$out_t6" "=== a/JOURNAL.md ==="
assert_contains "I7: level-b delimiter '=== a/b/JOURNAL.md ==='" "$out_t6" "=== a/b/JOURNAL.md ==="
assert_not_contains "I7: no absolute fixture paths leaked into delimiters" "$out_t6" "$T6/"

# -----------------------------------------------------------------------------
# Test 7 — error path: non-existent input
# Supports the I6 boundary; non-existent paths exit 2 to distinguish them
# from the "valid path with no journals" empty case.
# -----------------------------------------------------------------------------
printf 'Test 7 (error path, non-existent input):\n'
T7="$TMP_BASE/nonexistent-path"
bash "$WALK" "$T7" >/dev/null 2>"$TMP_BASE/t7.err"
ec_t7="$?"
err_t7="$(cat "$TMP_BASE/t7.err")"
assert_eq "error path: exit code is 2" "$ec_t7" "2"
assert_contains "error path: stderr names the missing path" "$err_t7" "path not found"

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
printf '\n'
printf 'PASS: %d   FAIL: %d\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
