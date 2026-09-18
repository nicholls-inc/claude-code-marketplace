# pr-swarm / review-triage — exact `gh` invocations

Every GitHub read and write both skills perform, copy-pasteable, verified
against `gh version 2.90.0` and a real PR (`owner/repo#68`, which
has real unresolved-at-time-of-check review threads, and `#70`, a real open
PR) on 2026-09-17. This file has ONE canonical copy, here at
`skills/pr-swarm/references/github-ops.md`; `review-triage`'s `SKILL.md`
cites it directly at
`${CLAUDE_PLUGIN_ROOT}/skills/pr-swarm/references/github-ops.md`. Do not
recreate a duplicate copy under `skills/review-triage/references/` — an
earlier duplicate was deliberately deleted because two copies of a command
reference drift silently.

Locally, every command on this page needs `dangerouslyDisableSandbox: true`
on the Bash call — `gh auth` reads the macOS Keychain and several of these
hit the network. In CI there is no sandbox; run the commands unchanged.

Substitute `$OWNER`, `$REPO`, `$PR` throughout. Where a command's real output
below shows `owner`/`repo`/a PR number, that is the verification
run, not a hardcoded value to leave in the skill.

**`$TMPDIR` is not set on GitHub's Linux runners** (`gh-actions`/`ubuntu-*`
images set neither `TMPDIR` nor any equivalent; only `RUNNER_TEMP` is
guaranteed). macOS sets `TMPDIR` for every process, which is why a local run
of every command below works unmodified and a CI run of the unmodified form
silently writes to `/pr-meta.json` at the filesystem root — root is not
writable by the job's user, so the first write fails with permission denied
(D21). Every command block below that reads or writes a `$TMPDIR` path
starts with:

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
```

Each code block in this file is written to be copy-pasteable as its own
Bash call — this line is repeated at the top of every one of them rather
than stated once, because a snippet copied on its own must not silently
regain the bug the moment it's lifted out of context.

**Two different scratch-file kinds, marked below, need two different
directories in CI.** Tag mode runs with `--permission-mode acceptEdits`,
which auto-allows the Write/Edit tools only *inside* `$GITHUB_WORKSPACE` and
denies them outside it. `$TMPDIR`/`$RUNNER_TEMP` is outside the workspace.

- **Redirect** — a `gh`/`jq`/`git` command's own stdout is captured with a
  plain shell `>` (or read back with `$(cat ...)`). This is Bash-tool output
  redirection, not the Write tool, so it is unaffected by `acceptEdits`'s
  workspace restriction; the `$TMPDIR` fallback above is correct and stays
  as-is for every command below marked **[redirect]**.
- **Write** — a scratch file whose *content the agent itself composes*
  (a finding body, a rendered summary, an assembled JSON array) rather than
  a command's raw output, and which the agent then creates with the Write
  tool rather than a shell redirect. A Write to a `$TMPDIR` path is denied
  outright under `acceptEdits`. These sites are marked **[write]** below and
  use `$SCRATCH_DIR` instead:

```bash
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
mkdir -p "$SCRATCH_DIR"
```

`$GITHUB_WORKSPACE` is set in every GitHub Actions job and unset locally, so
the fallback to `$PWD` (the repo root a local run is already invoked from)
keeps the same command working unmodified in both places, the same way the
`$TMPDIR`/`$RUNNER_TEMP` fallback does for the redirect sites. Run the
`mkdir -p` once per round before the first `[write]` site.

## Hard constraint — never approve

**Every review this plugin posts uses the comment-only review event. The
review event that grants a GitHub approval, and the `gh pr review` flag
(long or short form) that submits one, are forbidden anywhere in this
plugin.** The commands below use the comment-only event throughout on
purpose — do not "simplify" one into the approving form even when the
skill's own verdict field says its assessment is approve-shaped; a verdict
string is a comment, never a GitHub approval. See the hard constraint
sections in both `SKILL.md` files for the full rationale.

---

## 1. Resolve the PR and gather the diff

**[redirect]**

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
gh pr view "$PR" --repo "$OWNER/$REPO" \
  --json number,title,body,headRefName,headRefOid,baseRefName,author,url,files,additions,deletions,changedFiles,isDraft,state \
  > "$TMPDIR/pr-meta.json"
```

Verified (`#70`, trimmed to the fields that matter for baselining):

```json
{"base":"main","head_sha":"fa3284c222e71d65aebe76b6a30e25ec05562323","isDraft":false,"number":70,"state":"OPEN","url":"https://github.com/owner/repo/pull/70"}
```

(`--jq '{number, url, base: .baseRefName, head_sha: .headRefOid, state, isDraft}'`
appended for the trimmed view above — the full `--json` list is what you
actually want in `pr-meta.json` since `files`/`additions`/`body` etc. are
used downstream.)

If `state` is `MERGED` or `CLOSED`, stop — there is nothing to review or
triage.

**[redirect]**

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
gh pr diff "$PR" --repo "$OWNER/$REPO" > "$TMPDIR/pr.diff"
git log "$BASE...$HEAD_SHA" --oneline
```

Owner/repo, when not already known:

```bash
gh repo view --json owner,name --jq '{owner: .owner.login, name}'
```

Verified: `{"name":"repo","owner":"owner"}`

## 2. Existing comments (dedup input, and top-level comment triage units)

**[redirect]**

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
gh api "repos/$OWNER/$REPO/pulls/$PR/comments" --paginate > "$TMPDIR/existing-inline.json"
gh api "repos/$OWNER/$REPO/issues/$PR/comments" --paginate > "$TMPDIR/existing-issue.json"
```

Verified against `#68`: returns `[]` cleanly when there are no top-level
issue comments yet (a valid, expected result — not an error).

### Shape top-level comments into triage units (D13)

A review-thread fetch (section 3) never surfaces Greptile, CodeRabbit, or
Copilot's findings — those three post a single top-level PR comment, not a
review thread. `review-triage`'s unit set (`SKILL.md` Step 3) includes every
top-level comment from the REST list above, keyed `issuecomment:<id>`, for
exactly that reason: a thread-only fetch reconciles perfectly while
triaging none of them. Shape the raw list the same way section 3 shapes
review threads — truncate to 1500 chars at the `jq` layer (same reasoning:
a bot summary runs tens of KB, and unlike a thread there is no cheaper
first-page read to fall back to) and mark the automated signal:

**[redirect]**

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
jq '[.[]
  | {
      id: ("issuecomment:" + (.id | tostring)),
      author_login: .user.login,
      author_type: .user.type,
      created_at,
      html_url,
      body_head: (.body[:1500]),
      body_truncated: ((.body | length) > 1500),
      comments_walked: 1,
      comments_has_next_page: false,
      participants: [{
        login: .user.login,
        typename: .user.type,
        automated_signal: (
          if (.user.type == "Bot") then "bot-typename"
          elif (.user.login | test("\\[bot\\]$")) then "bot-suffix"
          elif (.user.login | ascii_downcase
                | test("github-actions\\[bot\\]|dependabot\\[bot\\]|greptile|veria|coderabbit|cursor|sonarcloud|codescene|sourcery|ellipsis|stamphog|posthog-code"))
            then ("bot-list:" + .user.login)
          elif (.body | contains("🤖 Automated comment by **pr-swarm**")) then "header:pr-swarm"
          elif (.body | contains("_🛡️ Authored by Crosscheck 🛡️_")) then "header:crosscheck"
          else "none"
          end
        )
      }],
      is_sticky_summary: (.body | contains("<!-- pr-swarm-summary -->"))
    }
]' "$TMPDIR/existing-issue.json" > "$TMPDIR/issue-comment-units.json"
```

`typename` here is REST's `.user.type` (`"Bot"` / `"User"` / `"Organization"`),
not GraphQL's `__typename` from section 3's query — the two happen to use
the identical string values for the cases this file cares about, but they
are different fields from different APIs; do not assume a code path that
reads one can read the other interchangeably. `participants` is a
one-element array (a REST issue comment has no nested `comments` connection
to hide a 21st reply behind, unlike a review thread), and `comments_walked:
1` / `comments_has_next_page: false` are constants for exactly that reason
— they are still emitted here, not left for `review-triage` to invent,
because a consumer told to read this fetch's fields verbatim (D26) cannot
also be told to synthesize two of them itself for one unit kind and not
another.

`--paginate` on the command in this section already walks every page of
the top-level comment list — that is the only pagination this unit type
has. Unlike a review thread, one REST list element *is* the whole comment;
there is no nested `comments` connection underneath it to hide a 21st reply,
which is exactly why `comments_walked`/`comments_has_next_page` are fixed
constants (`1`/`false`) here rather than derived from anything.
`participants[0].automated_signal` is the real gate — `triage-rubric.md`
§1's bot-login and body-signature lists are what it encodes, and the check
runs against the **full** body, never `.body[:200]`.

Re-run against `#70` on 2026-09-17. This PR has picked up two more top-level
comments since the D13 shape was first verified — four now, three of them
real Crosscheck-authored comments (this repo's real `hnipps`-run
`crosscheck-pr-review` history), which finally exercises the
`header:crosscheck` branch live, not just the human-comment negative case:

```json
[
  {
    "id": "issuecomment:5715679415",
    "author_login": "hnipps",
    "author_type": "User",
    "body_head": "## Shepherd response — the 7 findings from the review summary that were outside the diff\n\n...",
    "body_truncated": true,
    "comments_walked": 1,
    "comments_has_next_page": false,
    "participants": [{"login": "hnipps", "typename": "User", "automated_signal": "none"}],
    "is_sticky_summary": false
  },
  {
    "id": "issuecomment:5715777735",
    "author_login": "hnipps",
    "author_type": "User",
    "body_head": "## Trigger mode resolved — from the gitops manifests\n\n...",
    "body_truncated": true,
    "comments_walked": 1,
    "comments_has_next_page": false,
    "participants": [{"login": "hnipps", "typename": "User", "automated_signal": "header:crosscheck"}],
    "is_sticky_summary": false
  },
  {
    "id": "issuecomment:5716015605",
    "author_login": "hnipps",
    "author_type": "User",
    "body_head": "## Added the explicit sweep trigger (`4b8c17f`)\n\n...",
    "body_truncated": true,
    "comments_walked": 1,
    "comments_has_next_page": false,
    "participants": [{"login": "hnipps", "typename": "User", "automated_signal": "header:crosscheck"}],
    "is_sticky_summary": false
  },
  {
    "id": "issuecomment:5716265776",
    "author_login": "hnipps",
    "author_type": "User",
    "body_head": "Adversarial review found a hole in the sweep trigger this PR adds...",
    "body_truncated": true,
    "comments_walked": 1,
    "comments_has_next_page": false,
    "participants": [{"login": "hnipps", "typename": "User", "automated_signal": "header:crosscheck"}],
    "is_sticky_summary": false
  }
]
```

The first comment is genuinely plain human prose and correctly reads `none`;
the other three carry the Crosscheck signature further down their body and
correctly read `header:crosscheck` — the same signature, the same `User`
typename, the same account (`hnipps`, the invoking human, per
`triage-rubric.md` §1) that section 3's live sample shows for a Crosscheck
thread comment. A bot summary from Greptile/CodeRabbit/Copilot is still
unobserved live at this repo (the honest gap `bot-typename`/`bot-suffix`
still carries) — but the body-signature branch, the one D13 exists for and
D26 fixed, now has real positive evidence for this unit kind too, not just
the negative "doesn't misfire on a human" case the two-comment sample above
used to show.

### Refetch one top-level comment's full body before acting on it

Only when a unit classified actionable had `body_truncated == true`. There
is no thread node to refetch from for a top-level comment (unlike section
3's per-thread refetch) — refetch the single comment directly:

```bash
gh api "repos/$OWNER/$REPO/issues/comments/$COMMENT_ID" --jq '.body'
```

## 3. Fetch review threads (GraphQL)

Filter to unresolved, non-outdated threads and truncate bodies to 1500 chars
at the `jq` layer — bot reviews run tens of KB each, and re-fetching full
bodies every round is the largest context cost in the triage loop. Bot and
pr-swarm comments put their tag/severity/summary in the first few hundred
chars, so the 1500-char head is enough to classify; refetch the full body
only for the one thread you are about to act on (section 4 below).

**Both connections in this query are paginated, and both must be walked to
`hasNextPage: false` before the bucket counts this feeds mean anything.**
`reviewThreads(first:100)` can hide threads past the first 100 on a large
PR. Worse, `comments(first:20)` on each thread hides replies past the 20th —
and the comments walk is the dangerous one: a thread with 20 bot replies on
page 1 and a human's reply as the 21st comment reads, on an unpaginated
fetch, as bot-only. That flips a human-participated thread to
bot-classified and lets the swarm act on it (reply to it, resolve it) as if
no human had weighed in. Never trust a `reviewThreads`/`comments` count
without confirming both `pageInfo.hasNextPage` fields are `false`.

Walk `reviewThreads` with `gh api graphql --paginate`, which auto-follows a
`$endCursor: String` variable against a `pageInfo{ hasNextPage, endCursor }`
block — no hand-rolled cursor loop needed for this connection:

```bash
gh api graphql --paginate -f query='
  query($owner:String!, $repo:String!, $num:Int!, $endCursor:String) {
    repository(owner:$owner, name:$repo) {
      pullRequest(number:$num) {
        reviewThreads(first:100, after:$endCursor) {
          pageInfo { hasNextPage endCursor }
          nodes {
            id
            isResolved
            isOutdated
            comments(first:20) {
              pageInfo { hasNextPage endCursor }
              nodes {
                databaseId
                author { login __typename }
                body
                path
                line
              }
            }
          }
        }
      }
    }
  }' -F owner="$OWNER" -F repo="$REPO" -F num="$PR" \
  | jq -s '[.[].data.repository.pullRequest.reviewThreads.nodes[]
      | select(.isResolved == false and .isOutdated == false)
      | {
          id,
          path: .comments.nodes[0].path,
          line: .comments.nodes[0].line,
          author_login: .comments.nodes[0].author.login,
          author_type: .comments.nodes[0].author.__typename,
          first_comment_id: .comments.nodes[0].databaseId,
          body_head: (.comments.nodes[0].body[:1500]),
          body_truncated: ((.comments.nodes[0].body | length) > 1500),
          reply_count: ((.comments.nodes | length) - 1),
          comments_walked: (.comments.nodes | length),
          comments_has_next_page: .comments.pageInfo.hasNextPage,
          participants: ([.comments.nodes[]
            | . as $c
            | {
                login: $c.author.login,
                typename: $c.author.__typename,
                automated_signal: (
                  if ($c.author.__typename == "Bot") then "bot-typename"
                  elif ($c.author.login | test("\\[bot\\]$")) then "bot-suffix"
                  elif ($c.author.login | ascii_downcase
                        | test("github-actions\\[bot\\]|dependabot\\[bot\\]|greptile|veria|coderabbit|cursor|sonarcloud|codescene|sourcery|ellipsis|stamphog|posthog-code"))
                    then ("bot-list:" + $c.author.login)
                  elif ($c.body | contains("🤖 Automated comment by **pr-swarm**")) then "header:pr-swarm"
                  elif ($c.body | contains("_🛡️ Authored by Crosscheck 🛡️_")) then "header:crosscheck"
                  else "none"
                  end
                )
              }]
            | unique)
        }
    ]'
```

`automated_signal` is `review-triage`'s full 7-value enum (`triage-rubric.md`
§1 / `SKILL.md`'s INV-3 definition): `bot-typename`, `bot-suffix`,
`bot-list:<entry>`, `header:pr-swarm`, `header:crosscheck`, `header:<skill>`
(reserved for a future vendored reviewer's own header — none is wired into
this jq yet, since only pr-swarm and crosscheck post a body-signature today),
or `none`. The check runs against the participant's **full** `.body`, not a
truncated prefix — see the load-bearing observation below for why a
prefix-windowed check is a live defeat, not a theoretical one. `typename`
(not `type`) matches the field name both `SKILL.md` and `triage-rubric.md`
use for the participant's GraphQL `__typename`; a differently-named field
here is silently unreadable by the classifier that consumes this fetch's
output.

`--paginate` emits one JSON object per page, so pipe through `jq -s`
(slurp) and index into `.[].data...` — a single non-paginated response has
no outer array to slurp away, which is why sections without `--paginate`
elsewhere in this file use a bare `jq '...'`, not `jq -s '...'`.

Re-run against `#68` on 2026-09-17, walked to `hasNextPage: false` on the
outer connection (`--paginate` fetched every page; **14 threads total**
exist on this PR — resolved, outdated, and open, combined; this total is
fixed and does not change as threads get resolved. Of those 14, **7** are
unresolved and non-outdated at this check time, down from the **13**
unresolved-and-non-outdated recorded in the Verification record at the
bottom of this file for the *initial* check — the PR's threads have been
resolved between checks, not re-fetched with a different query):

```json
{
  "id": "PRRT_kwDOS0fdqs6iPNgX",
  "path": "wrapper/src/wrapper_proxy/session.py",
  "line": 1203,
  "author_login": "hnipps",
  "author_type": "User",
  "first_comment_id": 4008462511,
  "body_head": "**[MAJOR · fault-path] reap_sync checks is_request_overdue() only on _busy workers, so a wedged free worker is never reclaimed**\n\n...\n\nThis is the second half of the `release()` finding; either fix alone narrows the window, both together close it.\n\n_🛡️ Authored by Crosscheck 🛡️_",
  "body_truncated": false,
  "reply_count": 1,
  "comments_walked": 2,
  "comments_has_next_page": false,
  "participants": [
    {"login": "hnipps", "typename": "User", "automated_signal": "header:crosscheck"},
    {"login": "hnipps", "typename": "User", "automated_signal": "none"}
  ]
}
```

Every thread on this PR came back `comments_has_next_page: false` — none of
its threads happen to carry more than 20 replies, so this run cannot show
the flip case directly. Check `comments_has_next_page` on every thread
anyway; the day one PR has a 21-reply thread with no cursor walk in place
is the day this silently misclassifies it.

### Walk one thread's comments past page 1

Do this for every thread where the query above reports
`comments_has_next_page: true`, before classifying that thread as
bot-only. Same `--paginate` mechanism, this time against the thread node
directly:

```bash
gh api graphql --paginate -f query='
  query($id:ID!, $endCursor:String) {
    node(id:$id) {
      ... on PullRequestReviewThread {
        comments(first:20, after:$endCursor) {
          pageInfo { hasNextPage endCursor }
          nodes {
            databaseId
            author { login __typename }
            body
            path
            line
          }
        }
      }
    }
  }' -F id="$THREAD_ID" \
  | jq -s '[.[].data.node.comments.nodes[]]'
```

Verified live against `#68`'s `PRRT_kwDOS0fdqs6iPNgX` thread (which has only
2 comments, so this is a smoke test of the mechanism, not the flip case —
`--paginate` walked it to `hasNextPage: false` and returned exactly the 2
nodes GitHub reports for it). The `jq -s '[...]'` wrapper always emits an
array — even for a fully-slurped single page — never a bare count; bodies
truncated here to their first line for brevity, full call returns the
complete `body` on each node:

```json
[
  {
    "databaseId": 4008462511,
    "author": { "login": "hnipps", "__typename": "User" },
    "body": "**[MAJOR · fault-path] reap_sync checks is_request_overdue() only on _busy workers, so a wedged free worker is never reclaimed** ...",
    "path": "wrapper/src/wrapper_proxy/session.py",
    "line": 1203
  },
  {
    "databaseId": 4008663234,
    "author": { "login": "hnipps", "__typename": "User" },
    "body": "Accepted in 92cb80e. `reap_sync`'s free-worker loop gains `elif worker.is_request_overdue(): reason = \"overdue\"` ...",
    "path": "wrapper/src/wrapper_proxy/session.py",
    "line": 1203
  }
]
```

(`.[].data.node.comments.nodes[]` after slurping, wrapped in `[...]`, is an
array of 2 elements here — matches this thread's real `commentsTotal: 2`
from the un-filtered query above. If you only need the count, pipe this
result through `| length` separately; the command as written above does
not do that.)

Re-run this per-thread walk's `participants`/`automated_signal`
classification against the FULL comment list this produces, not against the
first-page `comments.nodes` from the section-3 query above once a thread is
known to have a next page — the first page alone is exactly the "20 bot
comments hide the 21st human comment" trap this section exists to close.
Re-record `comments_walked` (now the full count) and `comments_has_next_page:
false` on the row once this walk completes.

REST list endpoints (section 2, section 5's summary lookup) use `--paginate`
too, for the same reason: an unpaginated list silently truncates instead of
erroring, so a comment past the first page is invisible, not flagged
missing.

**Load-bearing observation from this real output:** the comment body itself
proves it was authored by Crosscheck (`_🛡️ Authored by Crosscheck 🛡️_`,
Crosscheck's own signature — see `triage-rubric.md` §1's body-signature
list), yet `author_login` is `hnipps` and `author_type` is `"User"` —
Crosscheck posts through the invoking human's own GitHub-authenticated
account — always; `crosscheck-pr-review` is a local, human-invoked skill and
never runs in CI here. `pr-swarm`/`review-triage` share that same
`User`-typename behaviour only for a **local** run; in CI they run under
`claude-code-action` and post as `claude[bot]` instead (`typename == "Bot"`,
already caught by the check above without needing the header) — see
`triage-rubric.md` §1 for both cases stated together. `author_type == "Bot"`
alone still cannot distinguish, in the local/Crosscheck case, "our own
automated comment posted via the invoking user's token" from "a human wrote
this" — the body-header/signature check is the only reliable signal there,
never skip it. `automated_signal` above resolves to `header:crosscheck` for
this comment, correctly.

**D35 — the jq is right and the run still got it wrong.** On 2026-09-17 a
dry run of this skill against `#24` reported "no existing automated comments
on the PR, so nothing was deduped — the three issue comments present are
human-authored". All three are `hnipps`/`User` and all three carry
`_🛡️ Authored by Crosscheck 🛡️_`; this section's jq classifies every one of them
`header:crosscheck`. The run did not misread the jq's output — it formed its
own opinion from the author column instead of running it. One of those three
comments is titled "Rebased onto `main` and renumbered to I15/I16", the exact
ground the run's own CRITICAL finding covers, so the dedupe it skipped was
not hypothetical.

Two things follow, both fixed in `SKILL.md`'s *automated-comment index*: a
top-level automated comment carries no `(path, line)` and is a dedupe source
anyway — indexed as `(null, null, source, concern)`, because the tuple shape
describes inline threads, not what coverage means — and the CI
tracking-comment carve-out is now excluded on what it says (a checklist and a
job link, no finding) rather than on its missing anchor, which is the
over-general reading that let three Crosscheck comments fall out of the
index. A run may report a PR as carrying no automated comments only when this
jq returned `none` for every one of them.

**D26, stated for the record:** an earlier version of this jq filter tested
only `.body[:200]` (the first 200 chars) for pr-swarm's own header, and had
no Crosscheck check at all. On this exact live comment the Crosscheck
signature sits at the *end* of a 2000+ char body, past that 200-char window
— so the earlier filter reported `automated: false` here, misreading a
Crosscheck-authored thread as human-participated and deferring it forever.
The fix is two things together, not one: check the **full** body (no
prefix window), and check it against the full signature list — never widen
the character window alone and call the gap closed, since the next reviewer
skill's header can be longer than whatever window replaces it (see both
`triage-rubric.md` §1 and the `SKILL.md` human-participation gates).

**D33 — the `none` branch, and how far it is proven.** The first real
execution of this plugin never exercised the human gate: every
human-account comment it saw was Crosscheck-signed, so all of them
classified `header:crosscheck` and the deferral path never ran. Re-checked
on 2026-09-17 against `#35`, which is the only PR in this repo carrying
genuinely human review-thread replies (`akhiljariwala-source`, `User`, no
`[bot]` suffix, no reviewer-skill signature). The shipped command above,
run verbatim, returns `[]` there — all nine of `#35`'s threads are
resolved, seven also outdated, so the `isResolved == false and isOutdated
== false` filter correctly drops them all. Re-run with that one filter
relaxed and nothing else changed, all nine classify as:

```json
[
  {"login": "hnipps", "typename": "User", "automated_signal": "header:crosscheck"},
  {"login": "akhiljariwala-source", "typename": "User", "automated_signal": "none"}
]
```

which is live evidence for the `none` branch on a **review-thread**
participant, and for the mixed shape that must defer: one Crosscheck-signed
participant and one genuine human in the same thread. What remains unproven
is the gate's *action*, not its input — no PR in this repo has a
human-participated thread that is unresolved and non-outdated, so no live
run has yet reached the point of declining to resolve or reply to one.
Re-verify that the first time this plugin runs on a PR where a person has
replied in an open thread, and treat the deferral branch as untested until
then.

### Refetch one thread's full body before acting on it

Only when a thread you've classified `actionable` had `body_truncated ==
true`. Refetch one thread at a time, only the one you're about to fix:

```bash
gh api graphql -f query='
  query($id:ID!) {
    node(id:$id) {
      ... on PullRequestReviewThread {
        isResolved
        comments(first:1) { nodes { body } }
      }
    }
  }' -F id="$THREAD_ID" --jq '.data.node.comments.nodes[0].body'
```

Re-run against `#68` on 2026-09-17 on a thread this jq filter actually flags
`body_truncated: true` (`PRRT_kwDOS0fdqs6iPNj8`, 2719 chars; head shown here
for brevity, full body returned by the real call):

```
**[MAJOR · test-coverage] `_memory_cap` drops MCP_MAX_SESSIONS from the static check on pooled manifests: absent or typo'd value fails open to 10**

Both reviewers land on this one independently.

**Spec-chain view (hellebuyck):**

`test_every_server_either_caps_sessions_or_is_an_explicit_exemption...
```

## 4. Post inline review comments (a single review, `event=COMMENT`)

Build the review body as JSON and POST it with `--input` rather than
repeated `-f comments[]=...` flags — more reliable for an arbitrary number
of comments than trying to express an array through raw-field flags.

**Unproven — schema/shape checked by hand, not executed against a live PR**
(see Verification record at the bottom of this file). Build the JSON with
`jq -n`, not a heredoc: a quoted heredoc (`<<'JSON'`) never expands
`$HEAD_SHA` (a literal `HEAD_SHA_HERE` placeholder is a bug, not an
example), and switching to an unquoted heredoc to fix that would instead let
a comment body containing a backtick or `` $( `` be executed as shell —
comment bodies come from finding text this skill did not choose. `jq -n
--arg`/`--argjson` keeps every value literal regardless of its content:

`comments.json` is itself a **[write]** file — see the paragraph below the
command — read it from `$SCRATCH_DIR`. `review.json` is this command's own
`jq -n` output, a plain **[redirect]**, so it stays in `$TMPDIR`:

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
jq -n \
  --arg commit_id "$HEAD_SHA" \
  --arg body "$(printf '> [!NOTE]\n> 🤖 Automated comment by **pr-swarm** — not written by a human\n\npr-swarm review complete. See inline comments.')" \
  --argjson comments "$(cat "$SCRATCH_DIR/comments.json")" \
  '{commit_id: $commit_id, event: "COMMENT", body: $body, comments: $comments}' \
  > "$TMPDIR/review.json"

gh api "repos/$OWNER/$REPO/pulls/$PR/reviews" \
  --method POST \
  --input "$TMPDIR/review.json"
```

The review `body` above carries the bot-identifier header required by
`SKILL.md`'s "Bot identifier — REQUIRED on every posted comment" section —
"the review body" is explicitly one of the four surfaces that section names,
distinct from the per-finding inline comment bodies in `comments.json` (each
of which already carries its own copy of the header per `SKILL.md` Step 6a's
body shape). Never drop this to the bare "review complete" string — a review
body without the header is indistinguishable from a human's own review at
the point a reader or `review-triage`'s classifier looks at it in isolation.

`$SCRATCH_DIR/comments.json` is the array of `{path, line, side, body}`
objects (or the range form below) assembled earlier in the round from the
synthesized findings — build each element the same `jq -n --arg`/`--argjson`
way, never by string-concatenating a finding's body text into a heredoc or a
raw `-f` flag. Assembling this array is content composition, not a command's
raw output — it is a **[write]** site, hence `$SCRATCH_DIR` and not
`$TMPDIR`: see the note at the top of this file on why tag mode's
`acceptEdits` permission mode denies a Write to a `$TMPDIR` path outright.

For a line range (multi-line comment), replace the single comment object
with `start_line` + `start_side` + `line` + `side` (the *end* line uses the
bare `line`/`side` keys, matching the single-line form):

```json
{"path": "src/app.py", "start_line": 40, "start_side": "RIGHT", "line": 45, "side": "RIGHT", "body": "..."}
```

`event` must always be the literal string `"COMMENT"` in that JSON — never
`"APPROVE"` or `"REQUEST_CHANGES"` (a request-changes review is also not
used by this plugin; it still gates on human review the same as approve
would, and the plugin's job is to comment, not to gate).

Verified real evidence that `event=COMMENT` (not approve) is exactly how
automated review comments already land in this repo — `#68`'s own review
list:

```bash
gh api repos/owner/repo/pulls/68/reviews --jq '.[] | {id, state, user: .user.login}'
```

```json
{"id":5201583311,"state":"COMMENTED","user":"hnipps"}
```

(`state: "COMMENTED"` is what the API reports back for an `event=COMMENT`
submission — confirms the round-trip.)

### Fallback: individual comments

**Unproven — not executed against a live PR** (see Verification record at
the bottom of this file). If the batched review above is awkward for a
given payload (very large comment count, or a 422 you want to isolate to
one comment), fall back to one POST per comment. `comment-body.md` is a
single finding's rendered body, composed content rather than a command's
output — a **[write]** site:

```bash
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
gh api "repos/$OWNER/$REPO/pulls/$PR/comments" \
  --method POST \
  -f commit_id="$HEAD_SHA" \
  -f path="$FILE_PATH" \
  -F line="$LINE" \
  -f side="RIGHT" \
  -f body="$(cat "$SCRATCH_DIR/comment-body.md")"
```

A `422` here means the anchor (`path`+`line`) is not part of the diff's
hunks — demote that finding into the sticky summary instead of retrying the
same anchor.

## 5. Sticky summary — find, then upsert

Find an existing summary comment by its HTML marker (`pr-swarm` uses
`<!-- pr-swarm-summary -->`):

```bash
gh api "repos/$OWNER/$REPO/issues/$PR/comments" --paginate \
  --jq '[.[] | select(.body | contains("<!-- pr-swarm-summary -->"))][0].id'
```

Verified against `#68`: returns nothing (empty result on the `--jq` filter)
when there is no existing summary yet — this is the expected first-run
state, not an error; fall through to "create" below.

`summary.md` is the whole rendered sticky-summary document — composed
content, a **[write]** site in both branches below:

**Found** (`id` non-empty) → update in place:

```bash
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
gh api "repos/$OWNER/$REPO/issues/comments/$COMMENT_ID" \
  --method PATCH \
  -f body="$(cat "$SCRATCH_DIR/summary.md")"
```

**Not found** → create once:

```bash
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
gh pr comment "$PR" --repo "$OWNER/$REPO" --body-file "$SCRATCH_DIR/summary.md"
```

**Unproven — not executed against a live PR** (see Verification record at
the bottom of this file): both the PATCH and the create form above.

Never call the "not found" branch when a `COMMENT_ID` was found — that is
exactly the "second summary comment on the same PR" defect both `SKILL.md`
files call out.

## 6. Resolve a thread (no reply)

```bash
gh api graphql -f query='
  mutation($thread_id:ID!) {
    resolveReviewThread(input:{threadId:$thread_id}) {
      thread { id isResolved }
    }
  }' -F thread_id="$THREAD_ID"
```

**Unproven — schema-introspected only, not executed against a live PR** (see
Verification record at the bottom of this file). `threadId` is the only
required input field on `ResolveReviewThreadInput` (confirmed via `gh api
graphql` schema introspection against this repo's GraphQL endpoint —
`resolutionReason` is an optional enum, not needed here).

## 7. Reply in a thread (bot/own threads only — never on a human thread)

`reply-body.md` is the reply's rendered text, composed content — a
**[write]** site:

```bash
SCRATCH_DIR="${GITHUB_WORKSPACE:-$PWD}/.pr-swarm-tmp"
gh api graphql -f query='
  mutation($thread_id:ID!, $body:String!) {
    addPullRequestReviewThreadReply(input:{pullRequestReviewThreadId:$thread_id, body:$body}) {
      comment { id url }
    }
  }' -F thread_id="$THREAD_ID" -f body="$(cat "$SCRATCH_DIR/reply-body.md")"
```

**Unproven — schema-introspected only, not executed against a live PR** (see
Verification record at the bottom of this file). Confirmed via schema
introspection: `AddPullRequestReviewThreadReplyInput` requires
`pullRequestReviewThreadId` and `body`; `pullRequestReviewId` is optional and
can be omitted (the mutation attaches the reply to the thread directly). Do
**not** substitute a bare `threadId` field name here — the resolve mutation's
input type uses `threadId`, this one uses `pullRequestReviewThreadId`; they
are different input types and the field names are not interchangeable.

`body` is passed with lowercase `-f`, not `-F`: `-F` type-coerces (`true`,
`false`, `null`, bare numbers) and reads any value starting with `@` as a
file path instead of literal text — a reply body that happens to start with
`@` (an at-mention) would silently become a file-read attempt instead of a
comment. `thread_id` is a plain opaque ID with no such risk, so `-F` is fine
there.

`review-triage`'s `triage-rubric.md` §5 governs *when* a reply is
permitted: bot/own threads only, never on a thread a human has participated
in.

## 8. Required-check status — read what you can, degrade honestly

Three signals, checked in this order, because this repo's real state shows
they disagree with each other:

```bash
# (a) statusCheckRollup with an isRequired flag, when GitHub can compute it
gh pr view "$PR" --repo "$OWNER/$REPO" \
  --json statusCheckRollup,mergeable,mergeStateStatus,isDraft \
  --jq '{mergeable, mergeStateStatus, isDraft, checks: [.statusCheckRollup[] | {name, status, conclusion, isRequired}]}'
```

Verified against `#70` — **every** check reports `isRequired: null`:

```json
{"checks":[{"conclusion":"SUCCESS","isRequired":null,"name":"determine-sha","status":"COMPLETED"},{"conclusion":"SUCCESS","isRequired":null,"name":"test","status":"COMPLETED"},{"conclusion":"SUCCESS","isRequired":null,"name":"build / build","status":"COMPLETED"},{"conclusion":"SKIPPED","isRequired":null,"name":"push","status":"COMPLETED"},{"conclusion":"SKIPPED","isRequired":null,"name":"update-gitops","status":"COMPLETED"}],"isDraft":false,"mergeStateStatus":"BLOCKED","mergeable":"MERGEABLE"}
```

`isRequired` being `null` for every check is not a bug in the command — it
means GitHub could not resolve requiredness from this call. Check the other
two sources before concluding "no required checks":

```bash
# (b) classic branch protection (legacy)
gh api "repos/$OWNER/$REPO/branches/$BASE/protection" \
  --jq '.required_status_checks.contexts'
```

Verified against `main` in `owner/repo`:

```json
{"message":"Branch not protected","documentation_url":"https://docs.github.com/rest/branches/branch-protection#get-branch-protection","status":"404"}
```

A `404` here means this repo uses rulesets, not classic protection — try (c)
before assuming nothing is required.

```bash
# (c) repository rulesets (current GitHub mechanism)
gh api "repos/$OWNER/$REPO/rulesets" --jq '.[] | select(.target=="branch") | {id, name, enforcement}'
gh api "repos/$OWNER/$REPO/rulesets/$RULESET_ID" \
  --jq '.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context'
```

Verified against `owner/repo`: an active `main` ruleset exists
(`id: 21050370`, `enforcement: active`), but its rule list is
`["deletion", "non_fast_forward", "pull_request"]` — **no
`required_status_checks` rule at all**. So for this specific repo, right
now, there is genuinely no positively-declared required-check set from any
of the three sources.

**Resolution when all three sources fail to name a required set** (this
repo's actual current state, per the `#70` and `main`-ruleset evidence
above): requiredness cannot be positively confirmed, so `checks_green` is
`false`, full stop — no fallback to "treat every non-`SKIPPED` check as
required." This is `review-triage SKILL.md`'s Step 6, precondition 2, applied literally:
"if required checks cannot be determined, this is `false`." State plainly in
the report which of the three sources were tried and that none named a
required set; do not arm auto-merge on this signal alone. (An earlier draft
of this file carried a permissive fallback here that inferred a required set
from the rollup and would have armed auto-merge on exactly this repo's
current state — that reading was rejected; see the spec's D4.)

If (a) *does* return `isRequired: true/false` values in a repo that has
classic branch protection or a `required_status_checks` ruleset rule, prefer
(a) directly — it is already merged/resolved by GitHub and needs no
fallback logic.

## 9. Arm auto-merge

### Pre-check: is auto-merge even allowed on this repo?

`--auto` below fails outright on a repo where the *auto-merge* feature
itself is off, whatever the PR's own state — check the repo setting once
per round before arming, so that failure is distinguished in the report
from a CODEOWNERS/required-review block.

**Not** `gh api "repos/$OWNER/$REPO" --jq '.allow_auto_merge'` — that bare
`repos/{owner}/{repo}` REST path cannot be safely allowlisted for the
workflow's Bash tool: every scoped-Bash glob that matches this read also
matches write paths sharing the same prefix (`repos/{owner}/{repo}/pulls/<n>/merge`
among them), so granting the read grants the write. Use the GraphQL query
instead, which the workflow already allowlists broadly for sections 3, 6 and
7's mutations, and ask for the same fact by its GraphQL name,
`autoMergeAllowed` — note this field is not exposed under `gh repo view
--json` in `gh` 2.90.0 (checked: it errors with "Unknown JSON field"), only
via `gh api graphql` directly:

```bash
gh api graphql -f query='
  query($owner:String!, $repo:String!) {
    repository(owner:$owner, name:$repo) {
      autoMergeAllowed
    }
  }' -F owner="$OWNER" -F repo="$REPO" --jq '.data.repository.autoMergeAllowed'
```

Verified against `owner/repo`: `true`. If this reads `false`,
report `auto_merge_allowed: false` and stop — arming will fail, and that is
a repo-configuration fact to surface, not an error to retry past.

### Precondition 6 — stale approvals dismissed (D22)

`--auto` outlives this run: it arms the PR, and GitHub performs the actual
merge whenever its own requirements next go green — possibly hours later, on
whatever HEAD is current *then*, not the HEAD this round reviewed. With the
`synchronize` trigger gone (D20), no later run of this plugin is ever alive
to notice a human's push in between and disarm. The one thing that keeps a
late push from merging unreviewed is the base branch dismissing a stale
*approving* review the moment a new commit lands — that setting is what
turns "a human approved this" into a claim that is still true when GitHub
acts on it. This plugin never grants its own approval (the hard constraint
above); this precondition is what makes the approval it depends on, and
never gives itself, actually hold.

Same two sources as section 8, same strict reading: unreadable, absent, or
unprotected all evaluate to `false`, never a permissive default — a repo
that does not positively declare stale-approval dismissal must not be armed
on the strength of an approval that could go stale a minute later.

```bash
# (a) classic branch protection (legacy)
gh api "repos/$OWNER/$REPO/branches/$BASE/protection" \
  --jq '.required_pull_request_reviews.dismiss_stale_reviews'
```

Verified against `main` in `owner/repo` — same 404 as section
8(b), for the same reason (this repo uses rulesets, not classic protection):

```json
{"message":"Branch not protected","documentation_url":"https://docs.github.com/rest/branches/branch-protection#get-branch-protection","status":"404"}
```

```bash
# (b) repository rulesets (current GitHub mechanism)
gh api "repos/$OWNER/$REPO/rulesets" --jq '.[] | select(.target=="branch") | {id, name, enforcement}'
gh api "repos/$OWNER/$REPO/rulesets/$RULESET_ID" \
  --jq '.rules[] | select(.type=="pull_request") | .parameters.dismiss_stale_reviews_on_push'
```

Verified against `owner/repo`: the same active `main` ruleset
from section 8 (`id: 21050370`) does carry a `pull_request` rule here — this
is the rule D4's section-8 check found absent for `required_status_checks`,
present but for a different rule type — and its
`dismiss_stale_reviews_on_push` parameter reads:

```json
false
```

**Resolution when this repo's real state above is what you get:**
`stale_approvals_dismissed` is `false` — a rule exists but does not dismiss
stale approvals, which is the same outcome as no rule at all for this
precondition's purpose. Auto-merge on this repo, right now, is doubly
unarmable: `checks_green` is `false` per section 8, and independently
`stale_approvals_dismissed` is `false` per this section — either one alone
already blocks arming. State plainly in the report which of the two sources
were tried and their raw values; do not arm on an inferred or assumed
dismissal setting.

If (a) returns a boolean (classic protection is configured), prefer it
directly, the same as section 8(a)/(b) — resolved output needs no ruleset
fallback.

```bash
gh pr merge "$PR" --repo "$OWNER/$REPO" --auto --squash
```

**Unproven — not executed against a live PR** (see Verification record at
the bottom of this file); this file's owner did not arm auto-merge on any
real PR to produce this section.

`--squash` is the default merge strategy; read the repo's `CLAUDE.md` /
`.claude/rules/` for an override before assuming squash everywhere
(`review-triage SKILL.md` Step 0, item 5).

`--auto` arms auto-merge without merging immediately — GitHub merges once
its own requirements (required reviews, required checks) are satisfied.
This is exactly the behaviour the spec requires: the run arms the gate, a
human's approval (a real GitHub approving review from a person, never from
this plugin) is what actually lets the merge through.

If the command errors, the exact error text is the report — the usual cause
is a CODEOWNERS requirement or a required-review rule, both human gates by
design. Do not retry with `--admin` to bypass it; that would defeat the
gate this plugin is built to respect.

To disarm (e.g. a later round finds a new actionable thread after arming):

```bash
gh pr merge "$PR" --repo "$OWNER/$REPO" --disable-auto
```

**Unproven — not executed against a live PR** (see Verification record at
the bottom of this file).

---

## Verification record

Read-only commands in sections 1-3, 5 (find), 8(a)/8(b)/8(c), 9's pre-check,
and 9's precondition-6 (a)/(b) above were each run against real PRs in
`owner/repo` on 2026-09-17: `#70` (open, no review threads, used
for PR metadata / required-checks / `autoMergeAllowed` shape, and — added
for D13 — the top-level-comment shaping jq in section 2, against its two
real, human-authored top-level comments) and `#68` (closed; 14 review
threads total, of which 13 were unresolved-and-non-outdated at initial check
time, down to 7 at re-check, used for the GraphQL thread fetch, refetch, and
existing-comments shape — the drop reflects threads genuinely being
resolved between checks, not a different query or a changed total). The
`main`-branch checks (8(b)/8(c), 9's precondition-6 (a)/(b)) were run
directly against the repo, not scoped to either PR. `#35` was added on the
same date for section 3's `automated_signal` — it is the only PR here with
genuinely human review-thread replies — see *D33* in section 3 for what that
run does and does not prove.

**D13 gap, stated honestly:** the section 2 top-level-comment jq now has
real live evidence for the `header:crosscheck` branch of `automated_signal`
(three of `#70`'s four top-level comments, confirmed above) and for `none`
not misfiring on genuine human prose (the fourth). What it still lacks is a
live third-party bot summary — neither `#68` nor `#70` carries a real
Greptile/CodeRabbit/Copilot comment at the time of this check, so the
`bot-typename`/`bot-suffix`/`bot-list:<entry>` branches are verified by
inspection of the jq logic and by section 3's identical branches (which
share real evidence via GitHub Apps generally, just not a top-level comment
specifically), not by a live top-level-comment round-trip. Re-verify those
branches against a live third-party bot summary the first time this plugin
runs on a PR that has one.

**Every mutating command in this file is unproven — schema-introspected or
API-doc-checked, never executed against a real PR** (each is marked inline
at its own section too): the two review-posting forms in section 4, both
the PATCH and create forms of the sticky summary in section 5, the resolve
mutation in section 6, the reply mutation in section 7, and the arm/disarm
commands in section 9. Sections 6 and 7 were checked by GraphQL schema
introspection against this repo's live endpoint (confirms exact mutation
and input-field names). Section 4's review-post shape additionally has real
evidence for its *effect*, short of executing it here: a real existing
`COMMENTED`-state review already present on `#68` confirms `event=COMMENT`
round-trips as `state: "COMMENTED"`. Sections 5 and 9 have no equivalent
introspection or round-trip evidence — this file's owner did not
post/patch/resolve/reply/arm/disarm anything on a real PR to produce any of
this. Treat every one of these as needing a live-run check before relying
on it (`SKILL.md`'s dry-run / low-risk-PR verification steps exist for
exactly this reason).
