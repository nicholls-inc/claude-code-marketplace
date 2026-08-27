#!/bin/zsh
# Run one retrospective review case against ev-energy/core.
# Usage: ./run_case.sh <pr_number> <output_dir> [model]
set -u
PR=$1
OUT=$2
MODEL=${3:-opus}
mkdir -p "$OUT"
cd "${CORE_REPO:-$HOME/repos/core}"
echo "You are reviewing a merged PR retrospectively as if it were open. Fetch the diff of PR #$PR in ev-energy/core with 'gh pr diff $PR'. Review it rigorously for correctness bugs, concurrency issues, backwards-compatibility breaks, and production risks — the standard the crosscheck byfuglien reviewer applies. Do NOT post any comments, do NOT modify anything. Output: a numbered list of findings, each with severity and a one-line failure scenario. If you find nothing serious, say so." | \
claude -p --model "$MODEL" \
  --allowedTools "Bash(gh pr diff:*),Bash(gh pr view:*),Read,Grep,Glob" \
  > "$OUT/review_$PR.txt" 2> "$OUT/review_$PR.err"
echo "exit:$? pr:$PR"
