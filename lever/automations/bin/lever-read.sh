#!/usr/bin/env bash
# lever-read.sh: fixed, read-only GitHub reads for lever automations.
#
# Why this exists: `Bash(gh api --method GET:*)` does not restrict the method (gh uses the LAST
# --method/-X flag), and mid-pattern wildcards such as `Bash(gh api repos/*/pulls/*/comments)` can
# match a different endpoint plus `-X DELETE`. Allowlist this script instead. Every subcommand takes
# exactly the arguments shown, validates them, and calls one fixed GET endpoint or one fixed
# GraphQL query. No argument reaches gh as a flag.
#
# Install: copy to .github/lever/automations/bin/lever-read.sh (keep it executable).
#   --allowedTools "...,Bash(.github/lever/automations/bin/lever-read.sh:*)"
#
# Usage:
#   lever-read.sh pr-review-comments <pr>      inline review comments (id, author, commit_id, path, line, body)
#   lever-read.sh pr-review-threads  <pr>      review threads with resolution state and comments
#   lever-read.sh issue-timeline     <issue>   timeline events (labeled/unlabeled/closed, actor, time)
#   lever-read.sh issue-comments     <issue>   comments with author_association
#   lever-read.sh trusted-comments   <issue>   only comments by OWNER, MEMBER or COLLABORATOR, or by claude[bot] /
#                                              github-actions[bot] (the automations' own records; [bot] logins can't be taken by users)
set -euo pipefail

die() { echo "lever-read: $*" >&2; exit 2; }

[ $# -eq 2 ] || die "usage: lever-read.sh <pr-review-comments|pr-review-threads|issue-timeline|issue-comments|trusted-comments> <number>"
cmd=$1 n=$2
[[ $n =~ ^[1-9][0-9]{0,9}$ ]] || die "not a PR/issue number: $n"

repo=${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}
[[ $repo =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "bad repository: $repo"
owner=${repo%%/*} name=${repo#*/}

case $cmd in
  pr-review-comments)
    gh api --paginate "repos/$repo/pulls/$n/comments?per_page=100" \
      --jq '.[] | {id, in_reply_to_id, author: .user.login, author_association, commit_id, original_commit_id, path, line, created_at, html_url, body}'
    ;;
  pr-review-threads)
    # shellcheck disable=SC2016  # $owner etc. are GraphQL variables
    gh api graphql -F owner="$owner" -F name="$name" -F number="$n" -f query='
      query($owner: String!, $name: String!, $number: Int!) {
        repository(owner: $owner, name: $name) {
          pullRequest(number: $number) {
            number state mergedAt headRefOid
            reviewThreads(first: 100) {
              pageInfo { hasNextPage }
              nodes {
                id isResolved isOutdated path line
                resolvedBy { login }
                comments(first: 30) {
                  nodes { databaseId author { login } createdAt url body commit { oid } originalCommit { oid } }
                }
              }
            }
          }
        }
      }'
    ;;
  issue-timeline)
    gh api --paginate "repos/$repo/issues/$n/timeline?per_page=100" \
      --jq '.[] | {event, actor: (.actor.login // .user.login // null), label: (.label.name // null), created_at, state_reason: (.state_reason // null), source_issue: (.source.issue.html_url // null)}'
    ;;
  issue-comments)
    gh api --paginate "repos/$repo/issues/$n/comments?per_page=100" \
      --jq '.[] | {id, author: .user.login, author_association, created_at, html_url, body}'
    ;;
  trusted-comments)
    gh api --paginate "repos/$repo/issues/$n/comments?per_page=100" \
      --jq '.[] | select(.author_association == "OWNER" or .author_association == "MEMBER" or .author_association == "COLLABORATOR"
             or (.user.type == "Bot" and (.user.login == "claude[bot]" or .user.login == "github-actions[bot]")))
           | {id, author: .user.login, author_association, created_at, html_url, body}'
    ;;
  *) die "unknown subcommand: $cmd" ;;
esac
