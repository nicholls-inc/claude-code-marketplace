#!/usr/bin/env bash
# shepherd.sh: the only GitHub writes (and the batched read) pr-shepherd's agent may make, as fixed commands.
#
# Why: `Bash(gh api graphql:*)` allows any mutation (mergePullRequest, deleteRef...), `Bash(gh api --method GET:*)`
# can be overridden by a second --method, and `Bash(git push origin HEAD:*)` matches `HEAD:main`. The workflow
# copies this file and pr-state.graphql to a read-only directory outside the workspace before the agent starts,
# and allowlists only that copy. Every subcommand takes exactly the arguments shown and validates them.
#
#   shepherd.sh state <pr>                         one batched read (pr-state.graphql): head, checks, threads, mergeability
#   shepherd.sh resolve-thread <thread-node-id>    resolveReviewThread, only on an open, opted-in PR of this repo
#   shepherd.sh reply <pr> <comment-id> <file>     reply in a review thread; body read from <file>
#   shepherd.sh push <pr>                          git push origin HEAD:refs/heads/<that PR's head branch>
#
# push refuses: fork PRs, closed PRs, PRs without SHEPHERD_LABEL, the default branch, protected branches,
# and any commit that changes a file under .github/ (other than bringing it in line with the default branch).
# It never force-pushes.
set -euo pipefail

die() { echo "shepherd: $*" >&2; exit 2; }
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
label=${SHEPHERD_LABEL:-shepherd}
[[ $label =~ ^[A-Za-z0-9._:\ -]{1,50}$ ]] || die "bad SHEPHERD_LABEL: $label"
repo=${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}
[[ $repo =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "bad repository: $repo"
owner=${repo%%/*} name=${repo#*/}

num() { [[ $1 =~ ^[1-9][0-9]{0,9}$ ]] || die "not a number: $1"; }

# Prints "<state> <isCrossRepository> <labeled> <headRefName>" for an open-PR check.
pr_facts() {
  gh pr view "$1" --repo "$repo" --json state,isCrossRepository,headRefName,labels \
    --jq "[.state, (.isCrossRepository|tostring), ([.labels[].name] | index(\"$label\") != null | tostring), .headRefName] | join(\" \")"
}

opted_in() {
  local state cross labeled ref
  read -r state cross labeled ref < <(pr_facts "$1")
  [ "$state" = OPEN ] || die "PR #$1 is $state"
  [ "$cross" = false ] || die "PR #$1 is from a fork; the shepherd only works on same-repo branches"
  [ "$labeled" = true ] || die "PR #$1 is not labeled $label"
  printf '%s\n' "$ref"
}

cmd=${1:-}
case $cmd in
  state)
    [ $# -eq 2 ] || die "usage: shepherd.sh state <pr>"; num "$2"
    gh api graphql -F owner="$owner" -F repo="$name" -F number="$2" -F query=@"$here/pr-state.graphql"
    ;;

  resolve-thread)
    [ $# -eq 2 ] || die "usage: shepherd.sh resolve-thread <thread-node-id>"
    [[ $2 =~ ^PRRT_[A-Za-z0-9_-]{1,100}$ ]] || die "not a review thread id: $2"
    # shellcheck disable=SC2016  # $id is a GraphQL variable
    facts=$(gh api graphql -F id="$2" -f query='
      query($id: ID!) { node(id: $id) { ... on PullRequestReviewThread {
        pullRequest { number repository { nameWithOwner } } } } }' \
      --jq '.data.node.pullRequest | "\(.repository.nameWithOwner) \(.number)"')
    read -r thread_repo pr <<<"$facts"
    [ "$thread_repo" = "$repo" ] || die "thread is not in $repo"
    opted_in "$pr" >/dev/null
    # shellcheck disable=SC2016
    gh api graphql -F id="$2" -f query='
      mutation($id: ID!) { resolveReviewThread(input: {threadId: $id}) { thread { id isResolved } } }'
    ;;

  reply)
    [ $# -eq 4 ] || die "usage: shepherd.sh reply <pr> <comment-id> <body-file>"
    num "$2"; num "$3"
    [ -f "$4" ] && [ ! -L "$4" ] || die "body file must be a regular file: $4"
    [ "$(wc -c <"$4")" -le 65536 ] || die "body file over 64 KiB"
    opted_in "$2" >/dev/null
    gh api --method POST "repos/$repo/pulls/$2/comments/$3/replies" -F body=@"$4" --jq '.html_url'
    ;;

  push)
    [ $# -eq 2 ] || die "usage: shepherd.sh push <pr>"; num "$2"
    ref=$(opted_in "$2")
    [[ $ref =~ ^[A-Za-z0-9._/-]{1,200}$ && $ref != *..* && $ref != -* ]] || die "refusing unusual branch name: $ref"
    default=$(gh repo view "$repo" --json defaultBranchRef -q .defaultBranchRef.name)
    [ "$ref" != "$default" ] || die "refusing to push to the default branch"
    protected=$(gh api "repos/$repo/branches/$ref" --jq '.protected')
    [ "$protected" = false ] || die "refusing to push to protected branch $ref"
    git fetch --quiet --no-tags origin "+refs/heads/$default:refs/remotes/origin/$default" "+refs/heads/$ref:refs/remotes/origin/$ref"
    # A .github/ path may differ from the PR head only where HEAD now matches the default branch (a base merge).
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      mine=$(git rev-parse -q --verify "HEAD:$f" 2>/dev/null || echo none)
      base=$(git rev-parse -q --verify "origin/$default:$f" 2>/dev/null || echo none)
      [ "$mine" = "$base" ] || die "refusing: this push changes $f; the shepherd never edits .github/"
    done < <(git diff --name-only "origin/$ref" HEAD -- .github)
    git push origin "HEAD:refs/heads/$ref"
    ;;

  *) die "usage: shepherd.sh <state|resolve-thread|reply|push> ..." ;;
esac
