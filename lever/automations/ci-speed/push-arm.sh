#!/usr/bin/env bash
# push-arm.sh: the only way the ci-speed lane pushes a branch it wrote by hand.
#
# Why: `Bash(git push origin ci-arm/*)` also matches `git push origin ci-arm/x:main` and
# `git push origin ci-arm/x HEAD:main`, because the wildcard spans spaces and refspecs. The workflow copies
# this file to a read-only directory outside the workspace before the agent starts and allowlists only that copy.
#
#   push-arm.sh ci-arm/<name>      git push origin HEAD:refs/heads/ci-arm/<name>   (never --force)
#
# Refuses any other ref, and any commit that changes a file under .github/ unless ALLOW_GITHUB_EDITS below is
# true. Set it to true only together with an L4 standing grant for CI-config arms (prompt GRANT), and only
# if the token may push workflow files; otherwise CI-config arms go to the goal issue as a patch.
set -euo pipefail

ALLOW_GITHUB_EDITS=false

die() { echo "push-arm: $*" >&2; exit 2; }
[ $# -eq 1 ] || die "usage: push-arm.sh ci-arm/<name>"
ref=$1
[[ $ref =~ ^ci-arm/[A-Za-z0-9._-]{1,80}$ && $ref != *..* ]] || die "refusing ref $ref: only ci-arm/<name>"

default=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name)
git fetch --quiet --no-tags origin "+refs/heads/$default:refs/remotes/origin/$default"
if [ "$ALLOW_GITHUB_EDITS" != true ] && [ -n "$(git diff --name-only "origin/$default...HEAD" -- .github)" ]; then
  die "refusing: this branch changes .github/ (deliver CI-config arms as a patch on the goal issue)"
fi
git push origin "HEAD:refs/heads/$ref"
