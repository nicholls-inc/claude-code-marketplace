#!/bin/zsh
# Usage: gen_case.sh <pr> <arm bare|delib|cc> [run_id]
set -u
PR=$1; ARM=$2; RUN=${3:-r1}
GEN=$(dirname "$0")
WT="$GEN/wt_${PR}_${ARM}_${RUN}"
SPEC=$(cat "$GEN/spec_${PR}.md")
case $ARM in
  bare)  DISCIPLINE="Implement the change directly." ;;
  delib) DISCIPLINE="Before writing any code, write a short design: restate the requirements in your own words, describe your planned approach and the data flow, and list the decisions you are making. Then implement according to that design. Do not use any verification tooling beyond ordinary reading of the code." ;;
  cc)    DISCIPLINE="Before writing any code: (1) enumerate the invariants and failure modes of the component you are changing — boundary conditions, empty inputs, type/relation mismatches, unit errors, error paths; write them as an explicit numbered contract. (2) Implement. (3) Before finishing, re-audit your diff line by line against each invariant and state for each how the code satisfies it, fixing anything that does not. This contract-first discipline is mandatory." ;;
esac
cd "$WT"
printf '%s\n\nTask specification:\n\n%s\n\nConstraints: work only in this checkout; do not run git commit/push; do not run the test suite (no DB available); edit source and test files as needed. When done, summarize what you changed.' "$DISCIPLINE" "$SPEC" | \
claude -p --model opus \
  --allowedTools "Read,Grep,Glob,Edit,Write,Bash(git diff:*),Bash(git status:*)" \
  > "$GEN/out_${PR}_${ARM}_${RUN}.txt" 2> "$GEN/out_${PR}_${ARM}_${RUN}.err"
RC=$?
git -C "$WT" diff > "$GEN/diff_${PR}_${ARM}_${RUN}.patch"
echo "exit:$RC pr:$PR arm:$ARM run:$RUN diffbytes:$(wc -c < "$GEN/diff_${PR}_${ARM}_${RUN}.patch")"
