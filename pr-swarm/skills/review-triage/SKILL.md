---
name: review-triage
description: >
  Triage, act on, and close out the review conversation on one pull request.
  Runs `pr-swarm` when the diff since the last reviewed SHA touches non-doc
  files, then buckets every unresolved review thread, every top-level
  comment and every unanchored finding as actioned / resolved / promoted /
  deferred / skipped, with pagination exhausted and counts that reconcile.
  Emits one evidence row per unit — participants, automated signals, and
  that unit's own comment-pagination state. Applies every non-ambiguous fix,
  resolves bot and own threads, defers anything a human touched, loops until
  quiet (5 rounds max) while waiting for CI and review bots, and arms
  auto-merge only on six preconditions read in the arming round — among them
  a branch-protection read that refuses to arm unless the base branch
  dismisses stale approvals on new commits, and a HEAD re-read after arming
  that disarms on mismatch. Every commit carries the fixed
  `PR-Swarm-Automated: true` trailer, which is how the CI workflow
  recognises its own pushes. Never submits an approving GitHub
  review. Use when the user says "triage this PR", "review-triage", "deal
  with the review comments", or "address the review feedback". Takes an
  optional PR number or URL, plus `--dry-run` for zero GitHub writes.
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Skill, AskUserQuestion
---

# Review Triage

One PR per invocation. This skill owns the *act and close out* half of
`pr-swarm`: it decides what to review, triages every open thread, applies the
fixes it is allowed to apply, loops until nothing actionable remains, and arms
auto-merge when — and only when — the gate below is satisfied. The review
itself belongs to the `pr-swarm` skill, which this skill invokes.

It runs unchanged locally and inside GitHub Actions. The only difference is the
mode (below), and unattended mode is strictly more conservative.

## Invariants — checked, not advised

These are not style guidance. Each one has a check, a place in the run where
the check executes, and a failure action. **Every invariant's outcome is
emitted in the final report (interactive) and in the structured result
(unattended) as a named field.** A run that cannot emit an invariant's field
has failed that invariant and must report `INVARIANT_VIOLATION`, not a clean
result.

Two of these invariants used to be self-attested — a run that skipped the
human gate, or armed auto-merge on three preconditions out of four, produced
output indistinguishable from a compliant run. INV-3 and INV-5 now require
**evidence in the artifact**: per-unit disposition rows with participants,
their automated signals and the row's own comment-pagination state, and
preconditions re-derived in the arming round with the command and its raw
output recorded verbatim. An invariant whose evidence is absent is failed,
whatever the run asserts about itself.

Both of those evidence requirements exist because an honest-looking run
defeated the earlier version. A run-level `pagination_exhausted` flag is a
claim about the fetch, not about the unit being acted on: one unwalked
comments page hides a human's 21st comment, that unit's participant list is
*truthfully* all-bot, the row passes every check, and a human's thread gets
resolved on the strength of it. Only per-row pagination state catches that.
Likewise `gh pr merge --auto` is SHA-agnostic: it arms the PR, not the
commit, so preconditions honestly recorded against SHA X merge SHA Y if
anyone pushes during the CI wait. A HEAD re-read *after* arming catches the
push that lands while the run is still alive — and nothing inside the run can
catch the push that lands after it exits, which is the larger window: `--auto`
survives the run, and GitHub merges whenever *its* requirements go green,
minutes or hours later, on whatever HEAD is current by then. That window is
closed from outside, by branch protection dismissing stale approvals on new
commits, which is why precondition 6 reads branch protection and refuses to
arm without it.

A later agent editing this skill may not remove an invariant, weaken its
check, or drop its report field without replacing it with a stronger check.
Deleting a row below is a spec change, not a refactor.

| ID | Invariant | Checked where | On failure |
| --- | --- | --- | --- |
| **INV-1 Never approve** | No run submits an approving GitHub review. Every posted review uses `event=COMMENT`. The approving review event, and the approve flag on `gh pr review`, are forbidden in any form — including as a step to satisfy a branch-protection rule. The strings themselves are absent from this plugin by design, so the Step 0 ban grep over **both** the plugin and the workflow must return nothing. | Before every GitHub write (Step 4, Step 6) and in the pre-flight grep of Step 0 | Abort the write, report `INVARIANT_VIOLATION: INV-1`, stop the run |
| **INV-2 Buckets reconcile** | Every **triage unit** ends in exactly one of five buckets: `actioned`, `resolved`, `promoted`, `deferred`, `skipped`. A unit is an unresolved non-outdated thread from the Step 3 fetch, **a top-level PR comment from the same fetch** (where Greptile, CodeRabbit and Copilot put their findings — a thread fetch never surfaces those), or an `anchored: false` finding returned by `pr-swarm`. `units == actioned + resolved + promoted + deferred + skipped`, no id in two buckets, and `pagination_exhausted == true` — an unpaginated fetch reconciles perfectly while ignoring thread 101. | End of Step 4, every round | Do not proceed to Step 6. Report the unaccounted ids explicitly and set `unresolved_actionable_remaining: true` |
| **INV-3 Human participation defers** | A unit with any participant that is not provably automated is deferred **untouched** — no fix pushed for it, no resolve, no reply. Unclassifiable participants count as human, **and a participant list that is not provably complete is unclassifiable**. **The evidence is the artifact:** the report carries one disposition row per unit, in every bucket, each row listing `participants[]` as `{login, typename, automated_signal}`, the row's own `automated_signal`, and the row's own `comments_walked` (how many comments were read and classified) and `comments_has_next_page` (whether any remain unread). For every row, `(any participant with automated_signal == "none" OR comments_has_next_page == true)` must coincide with `gate: "human"`, `bucket: "deferred"`, and `actions: []`. A row missing, missing its participants, or missing either pagination field, fails INV-3 regardless of what the run believes it did. **The participant list is re-derived at the pre-action refetch, not carried from assembly**, and the row records `participants_read_at` (that refetch's timestamp) — rows are built in Step 3 and acted on in Step 4, so a human who replies in between leaves every recorded field honest and the gate reading a list that is no longer true. | Step 4, immediately after the pre-action refetch and before any action on the unit; re-checked across all assembled rows in Step 7 | Treat as human, defer, record `classification_uncertain: true`. A row-level contradiction found in Step 7 is `INVARIANT_VIOLATION: INV-3` |
| **INV-4 Loop cap** | At most 5 rounds per invocation. `round` increments exactly once per pass, **at Step 2 entry**, and is never reset within an invocation. A pass that would take `round` above 5 does not run at all. | Step 2 entry | Abort the pass, stop looping, report `loop_exhausted: true` with what remains |
| **INV-5 Auto-merge gate** | Auto-merge is armed only when all five pre-arm preconditions — 1–4 **and 6** — evaluate `true` **from reads taken in Step 6 of the arming round itself**, with each boolean recorded next to the command that produced it, that command's raw output, the read timestamp, and the head SHA observed at read — which must equal the SHA being armed. A precondition carried over from an earlier round, or recorded as a bare boolean with no evidence, does not count as `true`. **Precondition 6 is the base branch's stale-approval rule**, read from branch protection: unless the base dismisses stale approving reviews when new commits are pushed, the arm is refused outright, because `--auto` outlives this run and the merge it authorises happens on a HEAD nobody here has seen. **Precondition 5 is evaluated after the arming command, not before it:** HEAD is re-read immediately after `--auto` returns and must still equal `head_sha_armed_against`; on any mismatch the run disarms with `--disable-auto` and reports `armed: false`. `gh pr merge --auto` arms the PR, not a commit, so without 5 the evidence describes one SHA and the merge happens on another, and without 6 the evidence describes a SHA that stops being HEAD after the run is over. | Preconditions 1–4 and 6: Step 6, immediately before `gh pr merge --auto`. Precondition 5: Step 6, immediately after it | Do not arm, or disarm if already armed. Report each non-`true` precondition by name, with whatever evidence was obtained |
| **INV-6 Ambiguity never terminates** | An ambiguous item is never a reason to end the run early or to ask a question in unattended mode. It goes to `deferred` and is reported. | Step 4 and Step 7 | N/A — ambiguity has exactly one outlet |
| **INV-7 Dry-run writes nothing** | With `--dry-run`, zero GitHub writes and zero pushes occur: no comment, no reply, no resolve, no `gh pr merge`, no `git push`. | Guard on every mutating command | Abort the command, report `INVARIANT_VIOLATION: INV-7` |
| **INV-8 Commit trailer** | Every commit this skill creates carries the trailer line `PR-Swarm-Automated: true`, verbatim, in the commit message's final trailer block — no exceptions, including a rebase result, an amend, and a commit made in a round that later fails another invariant. Verified by reading the commit back (`git log -1 --format=%B`) **before** the push. The CI workflow's self-push guard matches this string, not the pushing identity. | Immediately after every `git commit`, before the matching `git push` (Step 4) | Amend to add the trailer and re-verify. If the trailer still cannot be confirmed, **do not push**: report `INVARIANT_VIOLATION: INV-8`, bucket the unit `deferred` with `gate: "error"`, and stop the loop |

INV-1 restated, because it is the one an eager agent rationalises away: this
skill authors fixes. An agent that both authors a fix and approves it has
removed the only independent check on its own work. Arming auto-merge is
permitted precisely because human approval remains a real gate in front of the
merge — so arming must never be made to work by supplying that approval.

## Modes

Resolve the mode first; every later step branches on it.

- **interactive** — a human ran the skill locally. Narrate each step, may use
  `AskUserQuestion` when a genuine choice arises, print the human report in
  Step 7.
- **unattended** — CI, or any invocation carrying `--unattended` /
  `$CI` / `$GITHUB_ACTIONS` set. **Never call `AskUserQuestion`**: there is no
  one to answer, and a blocked prompt burns the job. Ambiguous items always
  defer (INV-6). End with the structured result in Step 7 and nothing after it.

Detect: unattended if `--unattended` is passed, or `${GITHUB_ACTIONS:-}` is
`true`, or `${CI:-}` is `true`. Otherwise interactive. Print the resolved mode
in the first narration line so a misdetection is visible immediately.

`--dry-run` is orthogonal to mode and may be combined with either (INV-7).

GitHub is the source of truth for every bucket decision, so both modes are
safely restartable: a re-run re-derives state from the PR rather than from
local memory.

## Narration

Before each step, emit one terse present-tense line. Interactive: print it.
Unattended: append it to the `narration` array in the structured result.

Format: `[triage] <step> — <what and why>`

```
[triage] step 0 — mode=unattended dry_run=false round=0/5
[triage] step 2 — round=1/5 diff since a1b2c3d touches src/app.py, running pr-swarm
[triage] step 3 — 11 threads + 2 top-level comments + 2 unanchored findings = 15 units (INV-2 denominator), pagination exhausted
[triage] step 4 — thread PRRT_x actioned: add missing await in src/app.py:42
[triage] step 4 — reconcile 15 = 4+5+1+3+2 (actioned/resolved/promoted/deferred/skipped)
[triage] step 6 — auto-merge NOT armed: checks_green=false (required set unconfirmed), stale_approvals_dismissed=true
[triage] step 6 — armed against 9f2c1ab, HEAD re-read 9f2c1ab — match, staying armed
```

A silent gap longer than ~30s is the failure mode. Narrate mid-step for
anything slow: the `pr-swarm` invocation, a push, a CI wait, a pagination walk.

## Step 0 — Pre-flight

1. Resolve mode and `--dry-run` (above). Initialise `round = 0`,
   `marker_sha = null`, `dispositions = []`, `pr_swarm_result = null`.
2. Local runs: `gh` and `git` need `dangerouslyDisableSandbox: true` — `gh`
   auth reads the macOS Keychain and `git push` needs network. In CI neither
   applies.
3. Confirm `gh auth status` succeeds and the repo resolves:
   `gh repo view --json nameWithOwner -q .nameWithOwner`. If either fails,
   stop and report — this skill cannot degrade into a no-GitHub mode.
4. INV-1 pre-flight ban grep:

   ```bash
   set +e
   grep -rniE 'event=APPROV[E]|pr review[^;|&]*--approv[e]|pr review[^;|&]*[[:space:]]-[[:alnum:]]*[a][[:alnum:]]*([[:space:]]|$)' \
     "${CLAUDE_PLUGIN_ROOT}" .github/workflows 2>/dev/null
   rc=$?
   ```

   These are the exact bytes `pr-swarm`'s `SKILL.md` quotes for the same
   check. One check, one spelling: two files describing one grep with two
   different regexes is a check nobody can verify, and the narrower spelling
   missed the bundled short flag (`gh pr review -ab`). Changing the pattern
   here is a coordinated change across both files in one commit.

   Read `rc`, not just the output — all three values mean something
   different:

   - **`rc == 1`** — no match. **This is the pass**, and it is the case an
     agent most often misreads: `grep` exits non-zero *because* it found
     nothing.
   - **`rc == 0`** — a match printed. `INVARIANT_VIOLATION: INV-1`: stop
     before touching GitHub.
   - **`rc == 2`** — at least one path in the list could not be read. The
     `2>/dev/null` has already swallowed the diagnostic that would say which,
     so establish the cause yourself: test whether `.github/workflows`
     exists. If it does not — the ordinary case, a run from outside the
     repo — the plugin half really was checked, and that is a **degradation
     to record**, never a clean check over both halves. If it does exist,
     something else in the path list was unreadable and the check itself is
     broken: fix it and re-run rather than proceeding. Either way, `rc == 2`
     with empty stdout is byte-for-byte indistinguishable from the pass on
     stdout alone, which is why `rc` is captured rather than inferred.

   Three things about that pattern are deliberate:
   - It covers the short flag as well as the long one. A bundle like `-ab`
     approves just as effectively as the spelled-out flag, and a check that
     only knows the long form bans the spelling rather than the act.
   - It uses POSIX classes, not `\n` escapes. `[^\n]` inside a bracket
     expression means "not a backslash and not the letter n" on GNU grep and
     something else on BSD grep, so the same check behaved differently in CI
     and locally — which is the worst possible property for a safety check.
   - It greps the **workflow directory too**, not only the plugin. The
     acceptance criterion names both; a grep that reads only half of what it
     claims to cover is a criterion that passes by omission. When
     `.github/workflows` is absent the plugin half still runs and `rc` comes
     back `2` — the degradation above, not a pass over both.
5. Read the repo's `CLAUDE.md` (and `.claude/rules/` if present) for merge
   policy, the allowed merge strategy, and any CODEOWNERS-gated paths. Carry
   those constraints into Steps 4 and 6. Missing files just mean a short brief.
6. **Record the unpushable paths.** In CI this skill pushes with a GitHub App
   installation token, and GitHub rejects at push time any change under
   `.github/workflows/` from a token without the `workflows` permission — the
   app token used here does not have it. So a fix in a workflow file cannot
   land, however correct it is. Set `workflow_paths_unpushable = true`
   unattended (leave it `false` interactive, where a human's credentials
   usually can push there) and carry it into Step 4: a unit whose fix would
   touch `.github/workflows/` is bucketed `deferred` with `gate: "error"` and
   the reason "token cannot push under .github/workflows/" **before** the edit
   is attempted. Deciding this up front rather than at the push is the whole
   point: discovered at the push, the run has already edited the tree, run the
   tests and made a commit it must now unwind. This bites on every
   workflow-touching PR — including the one that ships pr-swarm's own
   workflow.

## Step 1 — Resolve the PR and the diff baseline

Use `$ARGUMENTS` if it looks like a PR number or URL; otherwise resolve the PR
for the current branch. One call, and parse `owner/repo` out of `url` rather
than spending a second call:

```bash
gh pr view <PR_OR_EMPTY> --json number,url,headRefName,baseRefName,headRefOid,isDraft,state \
  --jq '{number, url, head_ref: .headRefName, base: .baseRefName, head_sha: .headRefOid, draft: .isDraft, state}'
```

Record: PR number, owner/repo, base, head ref, HEAD SHA, draft flag, state.

If `state` is `MERGED` or `CLOSED`, there is nothing to triage — report and
stop. This is a clean terminal condition, not a failure.

`marker_sha` is the SHA `pr-swarm` **actually reviewed** most recently in
*this* invocation — `pr_swarm_result.head_sha`, not the SHA that was passed to
it. `null` on round 1, set in Step 2 thereafter.

The distinction is load-bearing. `marker_sha` is the floor of the next round's
diff check, so setting it to the HEAD read at call time, while `pr-swarm`
reports having reviewed a different SHA, marks a range reviewed that nobody
reviewed: the next round diffs from the newer of the two, and whatever lies
between them is never looked at again. Anchoring the marker to the SHA the
review was performed against is safe in both directions — if the two differ,
the next round re-reviews the overlap rather than skipping a gap.

## Step 2 — Run `pr-swarm` when the diff warrants

**Increment `round` here, first thing, once per pass (INV-4).** If the
increment would take `round` above 5, this pass does not run: leave `round` at
5, stop the loop, and go to Step 7 with `loop_exhausted: true`. Rounds 1
through 5 run; a sixth does not exist.

Then run `pr-swarm` when **either**:

- `marker_sha` is `null` (first round), **or**
- `HEAD_SHA != marker_sha` **and** `git diff --name-only <marker_sha>..<HEAD_SHA>`
  contains at least one non-doc file — anything other than `*.md`, `*.txt`,
  `LICENSE`, or pure whitespace changes.

### Invocation — all four inputs are required

```
Skill("pr-swarm", args="<pr_number_or_url> --head-sha <HEAD_SHA> --round <round>[ --dry-run]")
```

PR ref, head SHA, round number, and the dry-run state (present or absent, and
absence is a positive statement that writes are allowed). Do not omit one and
let `pr-swarm` infer it — an inferred head SHA is how a verdict for the wrong
commit enters the gate. If an input cannot be supplied, do not call
`pr-swarm`: record it as a degradation and proceed with precondition 1 false.

**This call is synchronous, and that is load-bearing.** `pr-swarm` declares no
`context: fork` in its frontmatter, so the `Skill` tool expands it inline in
this conversation and its return contract is in hand when the call returns —
Step 3 onward reads `pr_swarm_result` immediately, and the round's own
bookkeeping (INV-4, `marker_sha`) assumes it. Do not give `pr-swarm`
`context: fork`, and do not reach for a background dispatch to "parallelise"
the review: a fork reports back as a task notification in a later turn, which
would leave this skill triaging a PR whose review has not landed.

**This skill dispatches no subagents of its own.** `Agent` is deliberately
absent from its `allowed-tools`: every reviewer runs inside `pr-swarm`, which
owns the dispatch mode for all of them (see *Dispatch mode* in
`${CLAUDE_PLUGIN_ROOT}/skills/pr-swarm/SKILL.md` — every `Agent` call there passes
`run_in_background: false` wherever the harness offers the parameter, and
treats its absence as a question to check rather than an answer). If a
future change to this skill needs a subagent, it adds `Agent` to
`allowed-tools` **and** states the same flag at the call site; inheriting the
harness default is what turned one parallel round into 25 minutes of wakeup
polling on PR #24.

### What comes back — a JSON object, not prose

`pr-swarm` returns exactly these fields; read them, do not re-parse its
comments:

| Field | Use here |
| --- | --- |
| `head_sha` | The SHA the review was performed against. **Reject the verdict if this is not the current HEAD** (re-read HEAD to compare). A verdict for a superseded SHA is not a verdict for this PR's current state |
| `round` | Must echo the round that was sent. A mismatch is a contract violation: record a degradation and treat the verdict as invalid |
| `verdict` | `APPROVE` \| `APPROVE_WITH_NITS` \| `REQUEST_CHANGES` \| `BLOCKED`. An assessment, never a GitHub review event. Feeds Step 6 precondition 1. Compare case-insensitively with `-` normalised to `_`; any value outside that set makes precondition 1 false |
| `router` | `{danger, confidence, delegations_issued}` — `danger` is `LOW`\|`MEDIUM`\|`HIGH`\|`CRITICAL`, `confidence` is `HIGH`\|`MEDIUM`\|`LOW` (an enum, not a number). Reported, not acted on |
| `findings[]` | `{file, line, severity, reviewer, body, anchored, comment_id?}`. Anchored findings become threads in Step 3. **`anchored: false` findings went to the sticky summary instead of a review thread, so no thread fetch will ever surface them — they enter Step 4 as units in their own right (below)** |
| `degradations[]` | Merged into this skill's `degradations` |

### `body` and `comment_id` are consumed, not decorative

- **`body`** is the finding's own prose — the text `pr-swarm` posted or would
  have posted. It is the *only* input the Step 4 rubric has for an
  unanchored finding: a thread unit can be re-read from GitHub, an unanchored
  one cannot, because it exists in no thread. The actionable / nit /
  ambiguous test is entirely a test of this prose. A finding that arrives
  without a usable `body` cannot be classified: record the contract violation
  as a degradation, bucket the unit `deferred` with `gate: "ladder"`, and do
  not guess a classification from `severity` alone.
- **`comment_id`** is the REST id of the review comment `pr-swarm` posted for
  an anchored finding. It maps a **thread unit back to the finding that
  created it**: match it against each thread's comment ids
  (`first_comment_id`, and the databaseIds from the per-thread comment walk),
  and the thread's disposition row can name the finding it came from, so a
  reader of the report can trace a resolve back to the review that asked for
  it. Attribution is what it is for, and attribution is what it can do.

  It does **not** dedupe unanchored findings, and any claim that it does is
  false by construction: only `anchored: false` findings become finding
  units, and an unanchored finding has no comment id — that is exactly what
  `anchored: false` means. There is no id to match, so the dedupe that
  matters for the unit set is the location one: `file:line` + reviewer, which
  keeps `pr-swarm` from filing a summary-routed finding that some other
  reviewer already raised as a thread at the same place. Treat that as the
  primary key for finding units, not a fallback, and keep it reviewer-scoped:
  two reviewers can legitimately file at one `file:line` and must not
  collapse into one unit.

`pr-swarm` posts its own inline comments and upserts its own sticky summary;
this skill does not duplicate that. After it returns, keep the object as
`pr_swarm_result` and set `marker_sha = pr_swarm_result.head_sha` — the SHA
reviewed, never the SHA passed. If that field is absent or unparseable, leave
`marker_sha` unchanged: an unmoved marker costs one redundant review next
round, a wrongly advanced one costs a review that was owed.

If skipping the review, narrate `pr-swarm: skip (no non-doc changes since <sha>)`
and keep the previous round's `pr_swarm_result` — its `head_sha` is what
decides whether its verdict is still usable in Step 6.

The non-doc rule is firm. Interactive mode may skip a qualifying diff only
after an `AskUserQuestion` confirmation; unattended mode may not skip at all.
"Review fatigue" is not grounds.

## Step 3 — Assemble the units (this set is the INV-2 denominator)

One fetch per round. Filter to unresolved, non-outdated threads and truncate
bodies at the jq layer — bot reviews run to tens of KB and re-fetching full
bodies every round is the largest context cost in this loop. The exact
GraphQL query, the truncation jq, and the per-thread refetch live in
`${CLAUDE_PLUGIN_ROOT}/skills/pr-swarm/references/github-ops.md`. Use them
verbatim; do not improvise flags.

### Pagination is not optional, and it is not cosmetic

The counts mean nothing until every page is in hand, at **both** levels:

1. `reviewThreads` — walk `pageInfo.hasNextPage` / `endCursor` until
   `hasNextPage` is `false`. A first page of 100 looks complete on any PR that
   has 101 threads.
2. **Each thread's `comments`** — walk that connection's `pageInfo` too. This
   one is the INV-3 hazard, not just a counting one: the human-participation
   gate reads *every* comment, so a thread whose 21st comment is a human's and
   whose first page holds 20 bot comments will be classified all-bot and
   auto-resolved on a human's words.
3. REST comment lists (`pulls/<n>/comments`, `issues/<n>/comments`) — always
   with `--paginate`.

Record `pagination_exhausted` as a boolean. If any page cannot be fetched,
`pagination_exhausted = false`: that is an INV-2 failure for the round — name
the connection that truncated, skip Step 6, and do not claim reconciled
counts.

**Record it per unit as well, and do not let the run-level flag stand in for
that.** Every unit carries `comments_walked` (how many of its comments were
actually read and classified) and `comments_has_next_page` (whether GitHub
says more exist). `pagination_exhausted` is *derived* from those — it is
`true` only when every unit reports `comments_has_next_page: false` and every
list-level walk completed — never asserted directly. The two are not
interchangeable: a run can walk every page of `reviewThreads` and every REST
list, honestly set the run-level flag, and still have one thread whose 21st
comment was never read. That thread's participant list is truthfully all-bot,
its row passes every other check, and a human's thread gets resolved. Only
the per-row fields see it, which is why INV-3 fails a row that omits them.

### The units

```
units = unresolved non-outdated review threads (paginated)
      + top-level PR comments (paginated)
      + pr_swarm_result.findings where anchored == false
```

**Thread units** are keyed by thread id (`PRRT_…`).

**Top-level comment units** are keyed `issuecomment:<id>` and come from the
REST issue-comments walk in `github-ops.md` §2
(`gh api "repos/$OWNER/$REPO/issues/$PR/comments" --paginate`). They are units
because that is where Greptile, CodeRabbit and Copilot put their findings —
a summary comment, not a review thread — so a thread-only fetch triages none
of them while reconciling perfectly. That fetch emits the same participant
and pagination shape as the thread fetch, so these rows are filled from it
like any other: `participants` as a one-element array, and `comments_walked:
1` / `comments_has_next_page: false` as constants it emits rather than values
invented here — a REST issue comment has no nested comments connection to
hide a 21st reply behind. Note that its `typename` is REST's `.user.type`,
not the GraphQL `__typename` the thread fetch reads; the two agree on the
strings this skill tests, and they are still different fields from different
APIs. They cannot be resolved (GitHub has no resolve for an issue
comment) and they get no reply comment of their own: a reply to a top-level
comment is a new top-level comment, and a loop that posts one per round is
how a PR acquires forty of them. **The run report is their record**, and it
is the only one — see the threadless-unit rule in Step 4.

Three exclusions, all by rule and all recorded rather than dropped:

- `pr-swarm`'s **own sticky summary** (the comment carrying
  `<!-- pr-swarm-summary -->`) is bucketed `skipped` with the reason "own
  sticky summary — its findings enter as `findings[]` units". Triaging it
  would count every finding twice.
- **The CI run's own progress-tracking comment** is bucketed `skipped` with
  the reason "claude-code-action tracking comment — this run's own progress
  checklist". In tag mode the action posts a top-level `claude[bot]` comment
  on **every** run and edits it as the job proceeds. It is not the sticky
  summary, so the marker above misses it, and it is not a third-party bot
  summary either — so without this rule the agent triages its own checklist,
  and INV-2's denominator grows by one more permanently-present unit on every
  run, forever. Identify it by author `claude[bot]` **plus** the tracking
  comment's checklist shape (task-list lines the action maintains) and the
  absence of the `pr-swarm` summary marker; when the identification is
  uncertain, still `skipped`, with the uncertainty recorded as the reason —
  skipping a real finding costs one round, triaging your own progress
  checklist costs every round.
- A bot summary whose every actionable point is already covered by a thread
  unit or a finding unit at the same `file:line` from the same reviewer is
  bucketed `skipped` with the covering ids as the reason. Bots that post both
  a summary and inline threads restate themselves; the thread is the
  actionable copy.

**Unanchored-finding units** are keyed `finding:<reviewer>@<file>:<line>` and
carry no thread, so they cannot be resolved either, and nothing is posted for
them: their disposition is recorded in the run report and nowhere else.
They are classified from their `body` (Step 2). These are the one unit kind
whose row fields are constructed here rather than read from a fetch — no
GitHub read produced them, since they exist only in `pr-swarm`'s return
value — so set `comments_walked: 1`, `comments_has_next_page: false`, and
the single `header:pr-swarm` participant directly.

Dedupe an unanchored finding against a thread unit by **`file:line` +
reviewer** — that is the only key available, because an unanchored finding
carries no `comment_id` by construction (Step 2). If a thread from the same
reviewer already covers that location, the thread is the unit and the finding
is not a second one. Two *different* reviewers' findings at one location stay
two units. `comment_id` still does its own job here, on the anchored side:
where a thread's comment ids contain a finding's `comment_id`, record that
finding id on the thread's disposition row, so the row says which finding it
answers. That is attribution, not deduplication — the anchored finding was
never going to be a unit.

Nothing may be dropped from the unit set silently — a unit you decide not to
touch is still a bucket entry (`deferred` or `skipped`), never an omission.

Before acting on a thread **in any way** — fixing it, resolving it, replying
in it — refetch that one thread (command in `github-ops.md`): its full body
if it was truncated, and always its current participant list and comment
pagination state, which Step 4's gate re-derives from this fetch rather than
from the row assembled here. Refetch one thread at a time, only the one you
are about to touch. A thread you decide not to touch needs no refetch; its
row is a record, not a licence.

## Step 4 — Triage and act

Classification rules, the autonomy ladder, and the bot-login list live in
`${CLAUDE_PLUGIN_ROOT}/skills/review-triage/references/triage-rubric.md`. The
control flow here is what is load-bearing.

For each unit, in this order.

**The human gate runs twice, and the second run is the authoritative one.**
At position 1 below it is a screen over the row assembled in Step 3, cheap
and early so that units destined to defer are never classified or worked on.
It runs again at the pre-action refetch (position 5), over the list that
refetch returns, and that is the run whose verdict decides whether the action
proceeds — a unit that passed the screen and fails the re-derivation is
deferred there and then, with whatever work was done for it discarded rather
than pushed. Both runs apply the rules below; only the second can be trusted
at the moment of writing, because only it read the thread as it is now.

**1. Human-participation gate (INV-3), screening pass.** Classify *every*
comment in the thread, not only the first, over the fully paginated comment
list. A comment is automated only if its author is a known bot account
(`__typename == "Bot"`, a login ending `[bot]`, or a login on the bot list in
`triage-rubric.md`)
**or** its body carries the `pr-swarm` bot-identifier header or the signature
of a known automated reviewer skill (the local marketplace crosscheck plugin's included — PRs carry
comments from earlier manual crosscheck runs, and without that signal they
defer forever). Our own comments post through a human-typed account, so the
header — not the account type — is what marks them ours.

Record, for the unit's disposition row, one entry per participant:

```
{"login": "greptile-apps[bot]", "typename": "Bot", "automated_signal": "bot-typename"}
{"login": "hnipps", "typename": "User", "automated_signal": "header:pr-swarm"}
{"login": "hnipps", "typename": "User", "automated_signal": "none"}
```

These three keys are what the canonical fetch in `github-ops.md` §3 emits
per participant — `login`, `typename` (the GraphQL `author.__typename`, not a
`type` you rename here), and `automated_signal`. Consume them as they arrive.
The query computes the signal over the **whole** comment body, not a leading
slice, because a body-signature check against a truncated body reports
`none` for a real automated comment whose signature sits past the cut — and
`none` is the value that decides the gate. If the fetch you get back is
missing `automated_signal`, or hands you a bare boolean instead, that is a
contract violation, not something to reconstruct inline: record the
degradation and gate every affected unit as human. Both fetches emit the
field today, for every unit kind, so this should never fire — keep it
anyway. It is the guard that catches the reference drifting back, and that
regression has already happened once: the thread fetch was corrected first
and the top-level-comment fetch kept a bare boolean, which would have
deferred every bot summary on the PR while the counts reconciled perfectly.
A guard whose value is that it stays silent is not a dead branch.

`automated_signal` is exactly one of `bot-typename`, `bot-suffix`,
`bot-list:<entry>`, `header:pr-swarm`, `header:crosscheck`, `header:<skill>`,
or `none`. **Per participant, the strongest signal that fired is emitted;
per row, the weakest across participants wins** — the row's own
`automated_signal` is `none` if any participant is `none`. Strength and
weakness answer different questions (what proved this author automated, versus
whether anyone here is unproven) and applying both at one level collapses the
gate.

**The list the gate acts on is derived at the pre-action refetch, not at
Step 3 assembly, and is stamped with `participants_read_at`.** Rows are
assembled in Step 3 and acted on here in Step 4, and a human can reply in
that gap — review a
70-thread PR and the gap is minutes. Every field recorded at assembly stays
honest while the thread it describes stops being all-bot, and the gate then
resolves a thread a human has joined on evidence that was true when it was
written. The skill already refetches one thread's full body immediately
before acting on it (Step 3, last paragraph); that refetch is where
participants are re-derived, from the same fetch, and both the participant
list and `comments_walked` / `comments_has_next_page` are overwritten with
what it returns. `participants_read_at` is that refetch's timestamp, and it
goes on the row so a reader can see the gate read the thread as it was at
action time. INV-5 grew a post-arm re-read for exactly this shape of defeat;
this is its mirror image, and a row whose `participants_read_at` predates its
own actions is an INV-3 failure.

If, on that re-derived list, any participant is `none`, the unit is
**human**: bucket it `deferred` with `gate: "human"` and `actions: []`,
untouched — abandoning the action even if the fix is already written, because
the point of the gate is that the human owns the outcome from the moment they
speak. No fix pushed on its behalf, no
resolve, no reply. A human who replied in a bot thread has joined the
conversation and owns its outcome. If you cannot confidently classify a
participant, record `automated_signal: "none"`, set
`classification_uncertain: true` on the row, and treat them as human.

**An incomplete comment list is an unclassifiable participant list.** Record
`comments_walked` and `comments_has_next_page` on the row from the fetch. If
`comments_has_next_page` is `true`, walk the rest first (the per-thread walk
in `github-ops.md`) and re-record both fields. If the walk cannot complete,
the row keeps `comments_has_next_page: true` and the unit is human-gated on
that basis alone — `automated_signal: "none"`,
`classification_uncertain: true`, `gate: "human"`, `deferred`, untouched —
whatever the comments you did read look like. The 20 bot comments you can see
say nothing about the 21st. That row also forces `pagination_exhausted =
false` for the round, which fails INV-2 and skips Step 6.

A top-level comment unit's participant is its single author, classified by the
same rules. A bot's summary comment is `Bot`/`bot-suffix`/`bot-list:<entry>`;
a colleague's "nice work" is `none`, which defers it untouched — correct, and
it does not block auto-merge (Step 6).

An unanchored-finding unit has one participant: `pr-swarm` itself
(`header:pr-swarm`). It is never human-gated.

**2. Staleness and self-reference → `skipped`.** Inline threads are already
filtered by `isOutdated=false`. Top-level comment units have no `isOutdated`
flag, so they need this check explicitly: scan the body for a referenced
commit SHA; if present and != HEAD, the review is stale — the bot will
re-post against the new HEAD. Bucket it `skipped` with the referenced SHA as
the reason. Also `skipped` here, per the Step 3 exclusions: our own sticky
summary, and a bot summary that only restates threads already in the unit
set. Prefer skipping to acting on a stale review. `skipped` is a real bucket
in the INV-2 identity, not an exemption from it: a stale unit that lands in
no bucket breaks the reconciliation it is still counted in.

**3. Classify** the remaining bot / own units as actionable, nit, or
ambiguous per `triage-rubric.md`.

**4. Check the unit's fix path is pushable.** Before any edit, decide
which files the fix would touch. If any of them is under `.github/workflows/`
and `workflow_paths_unpushable` is set (Step 0, item 6), the fix cannot land:
bucket the unit `deferred` with `gate: "error"` and the reason "token lacks
the `workflows` permission — a change under .github/workflows/ is rejected at
push time", and move on **without editing anything**. GitHub rejects such a
push from an App installation token that was not granted `workflows`, and it
rejects it at the push, after the edit, the test run and the commit. Finding
that out at the push means unwinding all three; knowing it here costs one
comparison. It is an `error` deferral rather than `skipped` because the work
is real and unfinished — it blocks auto-merge (Step 6, precondition 3), which
is right: the review asked for a change this run cannot make, so a human has
to.

**5. Refetch, re-derive the participant list, re-run the gate, then act by
class.** The refetch is the one described at the end of Step 3; its output
overwrites `participants`, `comments_walked`, `comments_has_next_page` and
`participants_read_at` on the row. If the re-derived list now contains a
`none`, stop here: `gate: "human"`, `deferred`, `actions: []`, and say in the
reason that a human joined between assembly and action. Otherwise:

- **Actionable** — apply the edit, run the repo's tests, commit
  (`fix: address <reviewer> <short description>`) **with the trailer below
  (INV-8)**, verify the trailer, push, then resolve the
  thread and reply in it with the commit SHA and a one-line description (a
  reply is permitted here: the thread is all-bot or our own). Bucket
  `actioned`. Never `--no-verify`, never `--force-push` — on a non-fast-forward
  push, fetch and rebase; on a conflict you cannot resolve cleanly, stop and
  bucket the unit `deferred` with `gate: "error"` and the exact error.
- **Nit** — resolve, with a one-line reason in the reply and the report
  ("intentional — <reason>" / "out of scope — follow-up" / "disagree —
  <reason>"). Bucket `resolved`.
- **Ambiguous** — run the autonomy ladder *before* deferring:
  - *Just do it* — unambiguously better, one sensible way to get there,
    reversible. Apply, push, resolve. Bucket `promoted`.
  - *Recommend and ask* — more than one reasonable solution. Make the call,
    apply, push, resolve — **and the report entry must state the alternative
    considered** ("did X because Y; alternative was Z, say the word and I'll
    switch"). Bucket `promoted`. These are listed individually in the report,
    never folded into a bare count, so the human can redirect cheaply.
  - *Stop and ask* — genuinely unclear which outcome is better, or the change
    would need a design decision. Bucket `deferred` with `gate: "ladder"`,
    thread left unresolved. This is the only ladder outcome that defers
    (INV-6: it does not terminate the run).

**Threadless units — the fix half only; the report is the record.** Resolving
and replying are thread operations, and two of the three unit kinds have no
thread: a top-level comment cannot be resolved, and an unanchored finding
exists in no conversation at all. For those the fix half is unchanged —
apply, test, commit with the trailer, verify, push — and **nothing is written
to GitHub to acknowledge it**. Their `actions` read `["fix:<sha>"]`, or `[]`
where no fix was warranted.

Do not record "noted in sticky summary". That acknowledgement cannot be
performed by this skill: `pr-swarm` is the sole writer of the sticky summary
and regenerates it wholesale each round, so there is nothing to append to and
anything appended would be overwritten — and posting the note as a comment
instead is exactly the top-level reply forbidden above, for the reason given
there. An `actions` entry naming a write that never happened is worse than no
entry at all, because a reader of the report takes it for evidence. The run
report is the record for these units, and it is sufficient: each one has a
disposition row carrying its classification, its reason, and its commit SHA.

Bucket them by exactly the same rules; a nit on a threadless unit is
`resolved` with its reason recorded, even though nothing was resolved on
GitHub's side, because the bucket names this skill's disposition of the unit,
not a GitHub state change.

Interactive mode may use `AskUserQuestion` at a genuine fork. Unattended mode
may not — the ladder's stop-and-ask rung is the deferral, full stop.

**6. Emit the disposition row (INV-3 evidence).** One row per unit, for every
bucket — not only the deferred ones, because "deferred rows look compliant" is
exactly the hole this closes:

```json
{"id": "PRRT_x", "kind": "thread", "bucket": "actioned",
 "location": "src/app.py:42", "classification": "actionable",
 "gate": null, "automated_signal": "bot-typename",
 "comments_walked": 3, "comments_has_next_page": false,
 "participants_read_at": "2026-09-17T11:58:12Z",
 "participants": [{"login": "greptile-apps[bot]", "typename": "Bot",
                   "automated_signal": "bot-typename"}],
 "actions": ["fix:9f2c1ab", "resolved", "replied"],
 "answers_finding": "byfuglien@src/app.py:42",
 "reason": "missing await"}
```

`kind` is `thread`, `top_level_comment`, or `unanchored_finding`. `gate` is
`null` unless the unit is deferred, in which case it is `human`, `ladder`, or
`error`. `actions` is the empty list for `deferred` and `skipped`.
`answers_finding` is the optional `comment_id` match from Step 3 — the
finding this thread was created from, present only where a `comment_id`
matched, and attribution only: it never merges two units.

`participants_read_at` is the timestamp of the refetch the participant list
was derived from, and on any row with a non-empty `actions` it must be
**later than the Step 3 assembly of that row and earlier than the first
action taken on it**. A row whose participants were read before its round's
assembly, or not re-read at all, has been gated on stale evidence and fails
INV-3 whatever it says about itself. Rows that were never acted on (`actions:
[]`) may carry the assembly-time read: nothing was done on their strength.

`comments_walked`, `comments_has_next_page` and `participants_read_at` appear
on **every** row, in every bucket, including the rows where the first two are
trivially `1` / `false`. They
are the per-unit half of INV-3's evidence and they are not inferable from the
run-level `pagination_exhausted` flag: that flag says the fetch finished, this
pair says *this unit's* participant list is complete. A row asserting
`automated_signal` other than `none` while carrying
`comments_has_next_page: true` is a contradiction, and Step 7 fails it.

**7. Reconcile (INV-2).** At the end of the round, assert
`len(units) == actioned + resolved + promoted + deferred + skipped`, that no
id appears twice, that every row carries both pagination fields, that
`pagination_exhausted` is `true` **as derived from those rows** (not asserted
independently), and that there is exactly one disposition row per unit. Emit
the five counts and the denominator (split into threads, top-level comments,
and unanchored findings) in the narration. On mismatch, name the unaccounted
ids, set `unresolved_actionable_remaining: true`, and skip Step 6 entirely
this round.

### Commit trailer — every commit, verbatim (INV-8)

Every commit this skill creates carries this line, and this exact line:

```
PR-Swarm-Automated: true
```

It goes in the commit message's final trailer block, on its own line, with
that capitalisation and that single space after the colon. Nothing is
substituted into it — it is a fixed string, not a template.

```bash
git commit -m "fix: address <reviewer> <short description>" \
           -m "<one line on why>" \
           -m "PR-Swarm-Automated: true"
git log -1 --format=%B | grep -qxF 'PR-Swarm-Automated: true' \
  || { echo "INVARIANT_VIOLATION: INV-8"; exit 1; }
```

Verify before the push, not after: an unpushed commit can be amended, a
pushed one has already armed the trigger it was meant to suppress.

**Why this is load-bearing, and why it is the trailer rather than the
author.** The CI workflow fires on `synchronize`, and this skill pushes to the
PR branch — so its own push would fire the workflow again. The workflow skips
a run whose head commit carries this trailer. It deliberately does **not**
test the pushing identity: who appears as pusher depends on which token
pushed (`GITHUB_TOKEN`, a PAT, an app installation token, a local user's
credentials), the answer differs between local and CI runs and changes when
the token changes, and guessing it wrong **fails open** — the guard stops
matching, every push starts a fresh unattended run, and that loop ends at a
merge nobody asked for. A trailer this skill writes itself is the one signal
that cannot drift out from under the guard.

Consequences of that, all of them binding:

- If several commits are pushed in one push, **each one** carries the
  trailer. A guard reading only the head commit still works; a guard reading
  every pushed commit also works. Neither breaks.
- If a rebase, cherry-pick, or amend rewrites a commit, re-verify the trailer
  on the rewritten commit before pushing. A rewrite that drops it is an INV-8
  failure, not a cosmetic loss.
- Never strip, reword, or abbreviate the trailer to tidy a message, and never
  add a second variant of it. The workflow matches one fixed string; a second
  spelling is a guard that silently matches nothing.
- Changing the string is a coordinated change across this skill **and** the
  workflow, in one commit. Changing it here alone disables the guard.
- `--dry-run` creates no commits, so INV-8 reports `n/a`.

## Step 5 — Loop until quiet

The round counter belongs to Step 2 (INV-4); this step decides only whether
there is another pass and what must settle first.

After a round that pushed commits, before deciding the PR is quiet:

1. Re-read HEAD (`gh pr view --json headRefOid`). If it moved, the next round's
   Step 2 diff check runs against the new HEAD.
2. **Wait for CI**: poll required checks until none are `pending`
   (`gh pr checks <N>` / the requirements read in `github-ops.md`). Bounded
   wait — poll at ~30s intervals up to 15 minutes, or the remaining CI job
   budget, whichever is shorter.
3. **Wait for third-party review bots**: give the known bots on the list in
   `triage-rubric.md` a bounded window (~5 minutes after the push, or until
   each has commented on the new HEAD) to post against the new HEAD, so their
   findings enter the next round rather than arriving after auto-merge is
   armed. A bot that does not report inside the window is noted in the report
   and the loop continues — waiting is bounded, never blocking.

Continue to the next round when the round changed anything (pushed a commit,
or new units appeared). Exit the loop when **any** of:

- nothing actionable remains and no new units appeared this round, **or**
- `round == 5` — the cap is reached, so no further pass starts. Stop with
  `loop_exhausted: true` and report what remains (INV-4), **or**
- the PR became `MERGED` or `CLOSED`, **or**
- INV-2 failed this round (report and stop; do not keep pushing on an
  unreconciled state), **or**
- interactive mode and the user interrupts — stop at the next checkpoint and
  print the report.

## Step 6 — Arm auto-merge (six preconditions, re-derived here)

Skipped entirely in `--dry-run` (INV-7) and skipped when INV-2 failed.

Take the five pre-arm reads — preconditions 1–4 **and 6** — **now, in this
round**, immediately before arming. A boolean inherited from an earlier round
does not count (INV-5). For each, record in the report: the command, its raw
output (trimmed, but not paraphrased), the read timestamp, the head SHA
observed at read, and the derived boolean. Then arm only if all five are
`true` — and evaluate precondition 5, below, only after the arming command
has returned.

The numbering is not the evaluation order, and that is deliberate: 5 is the
post-arm HEAD re-read and keeps its number from the round that introduced it,
so that reports and the sibling skill's references stay comparable across
versions. Read it as "1–4 and 6 before, 5 after".

Record the SHA all five pre-arm reads were taken against as
`head_sha_armed_against` before issuing the command. It is what precondition
5 compares to.

| # | Precondition | Read taken here |
| --- | --- | --- |
| 1 | `verdict_ok` — `pr-swarm`'s verdict for the SHA being armed is `APPROVE` or `APPROVE_WITH_NITS` | `pr_swarm_result.verdict`, valid only if `pr_swarm_result.head_sha` equals the SHA being armed. A verdict for an older SHA, a missing result, or an unrecognised verdict string ⇒ `false` |
| 2 | `checks_green` — every required check has concluded successfully; none failing, none pending | `gh pr checks <N>` plus the requiredness read in `github-ops.md`. **Strict reading: if the required set cannot be positively confirmed, this is `false`.** There is no permissive fallback — a repo that declares no required checks must not thereby arm on any state at all |
| 3 | `no_unresolved_actionable` — no unit is deferred for a reason that still needs work | The round's disposition rows: `false` if any row has `gate: "ladder"` or `gate: "error"`. **Rows with `gate: "human"` do not block**, and neither does `skipped` — see below |
| 4 | `not_draft` — the PR is not a draft | `gh pr view <N> --json isDraft` re-read here, not carried from Step 1 |
| 5 | `head_unchanged_after_arming` — HEAD is still `head_sha_armed_against` once `--auto` has returned | `gh pr view <N> --json headRefOid` re-read **after** the arming command. Any other value ⇒ `false` ⇒ disarm |
| 6 | `stale_approvals_dismissed` — the **base** branch's protection dismisses stale approving reviews when new commits are pushed | `gh api "repos/$OWNER/$REPO/branches/$BASE/protection" --jq '.required_pull_request_reviews.dismiss_stale_reviews'`, read here against `head_sha_armed_against`. `true` ⇒ `true`. Anything else — `false`, the field absent, no `required_pull_request_reviews` block, the branch unprotected, or a `403`/`404` because the token cannot read protection — ⇒ `false`. **Unreadable is `false`**, on the same strict reading as precondition 2: a rule you cannot see is a rule you cannot rely on. On a ruleset-based repo, read the effective rules for the base branch instead and derive the same boolean; if neither read resolves, `false` |

### Precondition 6 — the window this run cannot watch

Precondition 5 closes the window between the arming command and the end of
this run. It does not close the one that matters more, because `--auto`
outlives the run: it stays armed, and GitHub performs the merge when **its**
requirements go green, which may be minutes or hours after this process has
exited, on whatever HEAD is current by then. A colleague pushes a commit at
that point and it is merged having been reviewed by nobody — while every
field in our report remains literally true, because every one of them was a
statement about `head_sha_armed_against` and about nothing else. There is no
check we can run from inside the run that sees this. With `synchronize`
dropped from the workflow triggers, there is no later run of ours to notice
it and disarm either.

So the gate has to be held by something that is still there after we are
gone, and branch protection is the only such thing: a base branch that
dismisses stale approvals on new commits means that colleague's push
invalidates the approving review, and the merge stops until a human approves
again. That human approval is one this skill may never supply (INV-1). The
never-approve rule exists precisely so that a human check sits in front of
every merge we arm; precondition 6 is what makes that check real rather than
nominal, because without dismissal the approval can predate the code being
merged.

Hence: **refuse to arm** when the base branch does not dismiss stale
approvals. Not "arm and warn" — a warning in a report nobody is reading at
2am is not a gate. Report precondition 6 by name with the protection read
that produced it, and say plainly that the merge needs either a protection
change or a human merging it by hand. On a repo whose base branch is
unprotected, this precondition is permanently `false` and auto-merge is
permanently unreachable, which is the correct answer for an unattended agent
on an unprotected branch.

That is not hypothetical here. On `owner/repo` today the classic
protection endpoint `404`s and the active `main` ruleset's `pull_request`
rule reads `dismiss_stale_reviews_on_push: false`, so precondition 6 is
`false` — and precondition 2 is independently `false` because the repo
declares no required checks (D4's strict reading). Auto-merge is currently
unarmable on this repo for two separate reasons, either of which alone would
block it. Report both raw values rather than stopping at the first `false`: a
reader who fixes one and expects the merge to arm needs to know the other is
waiting behind it.

### Why human-gated deferrals do not block (and what still does)

A human-gated deferral is deferred *before* classification — the gate fires on
participation, so the unit was never assessed as actionable at all. Read
literally, one "nice work, thanks for the fix" comment from a colleague would
block the merge forever, which is the opposite of the intent: the human gate
exists to keep the agent's hands off a human's thread, not to hand any
commenter a permanent veto. Human approval is still a real gate in front of
the merge, and every human-gated row is listed by name in the report, so the
human who commented sees it before the merge happens.

What does block: a `ladder` deferral (assessed, genuinely ambiguous, needs a
decision) and an `error` deferral (assessed as actionable, the fix or push
failed). Those are unfinished work, and precondition 3 is `false` while either
exists.

### The verdict is never a GitHub approval

The verdict in precondition 1 is **this skill's own assessment, posted as a
comment**. It is not, and may never be replaced by, a GitHub approving review
(INV-1). If arming fails because branch protection requires an approving
review, that is the correct outcome: report it as a human gate and stop. Do
not approve to unblock, and do not switch to a plain `gh pr merge` to bypass
the gate.

Arm with the strategy the repo policy allows (default squash):

```bash
gh pr merge <N> --auto --squash
```

If the command errors, capture the exact error text in the report — the usual
cause is a CODEOWNERS gate or branch protection, both of which are human gates
by design.

### Precondition 5 — re-read HEAD after arming, and disarm on mismatch

`gh pr merge --auto` takes no SHA. It arms **the pull request**, and GitHub
merges whatever HEAD is when its own requirements go green — which may be
minutes or hours later. Every read above is therefore a statement about
`head_sha_armed_against` and about nothing else. A colleague pushing one
commit while the CI wait runs leaves all five pre-arm preconditions honestly
`true`, every piece of evidence in the report accurate, and a different
commit merged. That commit was never reviewed, never triaged, and never
gated. Precondition 5 catches that push while this run is still alive;
precondition 6 is what covers the same push arriving after it exits.

So, immediately after `--auto` returns:

```bash
gh pr view <N> --json headRefOid -q .headRefOid
```

- **Equal to `head_sha_armed_against`** — precondition 5 is `true`, the arm
  stands, and the report records both SHAs.
- **Anything else** — precondition 5 is `false`. Disarm at once:

  ```bash
  gh pr merge <N> --repo <OWNER/REPO> --disable-auto
  ```

  Report `armed: false` with `head_sha_armed_against` and the observed
  `new_head_sha` side by side, and treat the new HEAD as new work: if the loop
  has rounds left (INV-4), the next round's Step 2 diff check runs against it.
  If the cap is spent, stop with `loop_exhausted: true` and say plainly that
  an unreviewed commit landed after the evidence was taken.
- **The disarm command itself fails** — this is the one outcome worth
  escalating loudly. The PR is armed against evidence that no longer describes
  it. Record `INVARIANT_VIOLATION: INV-5` with the exact error text and both
  SHAs, and stop the run. Do not retry into a merge.

The report asserts `head_sha_armed_against == new_head_sha` whenever
`armed: true`. A report claiming an arm with those two fields unequal is an
INV-5 failure on its face, checkable by anyone reading the artifact without
re-running anything.

Any non-`true` precondition is reported by name, with the evidence obtained.
Never arm on five of six, and never arm on booleans that no read in this
round produced.

## Step 7 — Report

Both modes report every invariant's outcome, one disposition row per unit, and
counts that reconcile. Before emitting, re-check the assembled rows against
INV-3: any row with a `none` participant, **or with
`comments_has_next_page: true`**, that is not `deferred` / `gate: human` /
`actions: []` is `INVARIANT_VIOLATION: INV-3`; so is any row missing
`participants`, `comments_walked`, `comments_has_next_page`, or
`participants_read_at`; and so is any row with a non-empty `actions` whose
`participants_read_at` is not later than that row's assembly — an action
taken on a participant list read before Step 4's refetch is an action taken
on stale evidence. Then re-check INV-5: if `automerge.armed` is `true`,
`head_sha_armed_against` must equal `new_head_sha`, **and** the
`stale_approvals_dismissed` precondition must be recorded `true`.

**Interactive** — one summary line, then the detail:

```
[triage] done — pr=#123 sha=<short> rounds=2/5 pr-swarm=ran units=15 \
  actioned=4 resolved=5 promoted=1 deferred=3 skipped=2 automerge=armed
```

Under it, in this order:

1. **Deferred items** — one line each: author, `file:line`, one-line reason,
   and the gate (`human` / `ladder` / `error`), with whether it blocks
   auto-merge.
2. **Recommend-and-ask decisions** — one line each: what was done, why, and
   the alternative considered. Never collapsed into the `promoted` count.
3. **Disposition table** — every unit: id, kind, bucket, `file:line`,
   classification, `automated_signal`, participants, `comments_walked`,
   `comments_has_next_page`, `participants_read_at`, actions. This is the
   INV-3 evidence; a summary count is not a substitute for it, and neither is
   the run-level `pagination_exhausted` flag.
4. **Auto-merge line** — armed, or each non-`true` precondition by name, each
   with the command and raw output it was derived from in this round. When
   armed, print `head_sha_armed_against` and the post-arm `new_head_sha` on
   that line, so the reader sees they match without opening the JSON.
5. **Invariant line** — `INV-1..INV-8: ok`, or each violated ID with detail.

**Unattended** — end with exactly this JSON block and nothing after it:

```json
{
  "pr": 123,
  "mode": "unattended",
  "dry_run": false,
  "rounds_used": 2,
  "loop_exhausted": false,
  "head_sha_in": "<HEAD when the run started>",
  "new_head_sha": "<HEAD at the end of the run; also the post-arm re-read>",
  "pr_swarm_ran": true,
  "pr_swarm_marker_sha": "<pr_swarm_result.head_sha — the SHA actually reviewed>",
  "pr_swarm_result": {
    "head_sha": "<SHA the review was performed against>",
    "round": 2,
    "verdict": "APPROVE_WITH_NITS",
    "router": {"danger": "MEDIUM", "confidence": "HIGH", "delegations_issued": 3},
    "findings_total": 7,
    "findings_unanchored": 2
  },
  "pagination_exhausted": true,
  "units_total": 15,
  "units_threads": 11,
  "units_top_level_comments": 2,
  "units_unanchored_findings": 2,
  "actioned": 4,
  "resolved": 5,
  "promoted": 1,
  "deferred": 3,
  "skipped": 2,
  "buckets_reconcile": true,
  "unresolved_actionable_remaining": false,
  "dispositions": [
    {"id": "PRRT_x", "kind": "thread", "bucket": "deferred",
     "location": "src/app.py:42", "classification": "not-assessed",
     "gate": "human", "automated_signal": "none",
     "comments_walked": 4, "comments_has_next_page": false,
     "participants_read_at": "2026-09-17T11:58:12Z",
     "participants": [
       {"login": "greptile-apps[bot]", "typename": "Bot", "automated_signal": "bot-typename"},
       {"login": "octocat", "typename": "User", "automated_signal": "none"}
     ],
     "actions": [], "reason": "human participated — awaiting author reply",
     "blocks_automerge": false},
    {"id": "PRRT_y", "kind": "thread", "bucket": "resolved",
     "location": "src/db.py:17", "classification": "nit",
     "gate": null, "automated_signal": "bot-suffix",
     "comments_walked": 23, "comments_has_next_page": false,
     "participants_read_at": "2026-09-17T11:58:41Z",
     "participants": [
       {"login": "coderabbitai[bot]", "typename": "Bot", "automated_signal": "bot-suffix"}
     ],
     "actions": ["resolved", "replied"],
     "reason": "intentional — the index is bounded by the loop above",
     "blocks_automerge": false},
    {"id": "issuecomment:2317745501", "kind": "top_level_comment",
     "bucket": "actioned", "location": "general",
     "classification": "actionable", "gate": null,
     "automated_signal": "bot-suffix",
     "comments_walked": 1, "comments_has_next_page": false,
     "participants_read_at": "2026-09-17T11:59:02Z",
     "participants": [{"login": "greptile-apps[bot]", "typename": "Bot", "automated_signal": "bot-suffix"}],
     "actions": ["fix:9f2c1ab"],
     "reason": "summary flagged an unguarded index; no inline thread for it",
     "blocks_automerge": false},
    {"id": "issuecomment:2317745999", "kind": "top_level_comment",
     "bucket": "skipped", "location": "general",
     "classification": "not-assessed", "gate": null,
     "automated_signal": "header:pr-swarm",
     "comments_walked": 1, "comments_has_next_page": false,
     "participants_read_at": "2026-09-17T11:57:50Z",
     "participants": [{"login": "hnipps", "typename": "User", "automated_signal": "header:pr-swarm"}],
     "actions": [],
     "reason": "own sticky summary — its findings enter as findings[] units",
     "blocks_automerge": false},
    {"id": "finding:byfuglien@src/app.py:88", "kind": "unanchored_finding",
     "bucket": "promoted", "location": "src/app.py:88",
     "classification": "ambiguous", "gate": null,
     "automated_signal": "header:pr-swarm",
     "comments_walked": 1, "comments_has_next_page": false,
     "participants_read_at": "2026-09-17T11:59:20Z",
     "participants": [{"login": "pr-swarm", "typename": "User", "automated_signal": "header:pr-swarm"}],
     "actions": ["fix:9f2c1ab"],
     "reason": "recommend-and-ask: extracted the guard into a helper",
     "blocks_automerge": false}
  ],
  "commits": [
    {"sha": "9f2c1ab", "trailer_verified": true,
     "subject": "fix: address greptile add missing await"}
  ],
  "recommend_and_ask": [
    {"location": "src/app.py:88", "did": "extracted the guard into a helper",
     "because": "two callers needed it", "alternative": "inline the check twice"}
  ],
  "automerge": {
    "armed": false,
    "round": 2,
    "head_sha_armed_against": "<SHA the five pre-arm reads were taken against>",
    "disarmed": false,
    "disarm_error": null,
    "preconditions": [
      {"name": "verdict_ok", "value": true,
       "source": "pr_swarm_result.verdict @ head_sha <SHA>",
       "raw": "APPROVE_WITH_NITS", "read_at": "2026-09-17T12:00:01Z",
       "head_sha_at_read": "<SHA>"},
      {"name": "checks_green", "value": false,
       "source": "gh pr checks 123 + requiredness read in github-ops.md",
       "raw": "required set could not be positively confirmed",
       "read_at": "2026-09-17T12:00:03Z", "head_sha_at_read": "<SHA>"},
      {"name": "no_unresolved_actionable", "value": true,
       "source": "dispositions: 0 rows with gate ladder|error",
       "raw": "deferred=3 (human=3, ladder=0, error=0)",
       "read_at": "2026-09-17T12:00:03Z", "head_sha_at_read": "<SHA>"},
      {"name": "not_draft", "value": true,
       "source": "gh pr view 123 --json isDraft",
       "raw": "{\"isDraft\":false}", "read_at": "2026-09-17T12:00:04Z",
       "head_sha_at_read": "<SHA>"},
      {"name": "stale_approvals_dismissed", "value": true,
       "source": "gh api repos/OWNER/REPO/branches/main/protection --jq .required_pull_request_reviews.dismiss_stale_reviews",
       "raw": "true", "read_at": "2026-09-17T12:00:05Z",
       "head_sha_at_read": "<SHA>"},
      {"name": "head_unchanged_after_arming", "value": null,
       "source": "not evaluated — arming was not attempted (checks_green false)",
       "raw": null, "read_at": null, "head_sha_at_read": null}
    ],
    "error": null
  },
  "invariants": {
    "INV-1_never_approve": "ok",
    "INV-2_buckets_reconcile": "ok",
    "INV-3_human_gate": "ok",
    "INV-4_loop_cap": "ok",
    "INV-5_automerge_gate": "ok",
    "INV-6_ambiguity_non_terminal": "ok",
    "INV-7_dry_run": "n/a",
    "INV-8_commit_trailer": "ok"
  },
  "degradations": ["greptile did not report within the 5m window"],
  "narration": ["[triage] step 0 — mode=unattended dry_run=false round=0/5"]
}
```

`unresolved_actionable_remaining` is `true` whenever a unit that would be
actionable could not be safely auto-fixed (`gate: "error"`), or INV-2 failed —
the caller needs to know the actionable state is not clean. A human-gated
deferral alone does not set it.

`new_head_sha` is the last HEAD read of the run, and when arming was attempted
it **is** the post-arm read of precondition 5. That makes the INV-5 assertion
a plain comparison of two fields any reader can make:
`automerge.armed == true` requires
`automerge.head_sha_armed_against == new_head_sha`. `disarmed` records that
`--disable-auto` was issued because they differed, and `disarm_error` carries
the exact error text if that command failed — the one case that reports
`INVARIANT_VIOLATION: INV-5`, because the PR is then armed against evidence
that no longer describes it. `head_unchanged_after_arming` is `null`, not
`false`, when arming was never attempted: it is a statement about an arm that
happened, and reporting it as `false` would hide which of the five pre-arm
preconditions actually stopped the merge. `stale_approvals_dismissed` is
never `null` — it is read before arming, like 1–4, so it always has a value,
and `false` there means the arm was refused rather than deferred.

## Graceful degradation

A missing dependency downgrades and warns; it never aborts the run. Every
degradation is listed in the report's `degradations`.

- **`pr-swarm` unavailable, or one of its four inputs unobtainable** — warn,
  skip Step 2's review, still triage existing threads. There is then no
  `pr_swarm_result`, so precondition 1 is `false` and auto-merge does not arm,
  and no unanchored-finding units exist this round.
- **`pr-swarm` returns prose, a missing field, or a `head_sha` that is not
  current HEAD** — treat the verdict as absent (precondition 1 `false`),
  record the contract violation as a degradation, and triage the threads
  anyway.
- **A reference file missing** — warn, fall back to the rules stated inline in
  this skill, and treat any unit you cannot confidently classify as
  ambiguous (which, unattended, means deferred).
- **A finding arrives without a usable `body`** — it cannot be classified, so
  do not classify it: record the contract violation as a degradation, bucket
  the unit `deferred` with `gate: "ladder"`, and report it as unfinished. A
  guess from `severity` alone is how a nit gets pushed as a fix.
- **The issue-comments walk fails** — the top-level comment units cannot be
  assembled, so the unit set is incomplete: `pagination_exhausted = false`,
  INV-2 fails for the round, do not arm. Triage the threads anyway and name
  the missing connection. Silently dropping this class is the exact defect
  that let bot summaries go untriaged while the counts reconciled.
- **A pagination cursor cannot be followed** — `pagination_exhausted = false`,
  which fails INV-2 for the round: report the truncated connection, do not
  claim reconciled counts, do not arm. Every unit whose own comment list is
  the one that truncated additionally defers on its own row
  (`comments_has_next_page: true`, `gate: "human"`), because an incomplete
  participant list cannot be shown to be all-bot.
- **CI signal unreadable, or requiredness unconfirmed** — `checks_green =
  false`; do not arm. Continue and report.
- **Branch protection unreadable, absent, or not dismissing stale approvals**
  — `stale_approvals_dismissed = false`; do not arm. Continue and report,
  naming which of the three it was, because the remedies differ: a token
  permission to fix, a branch to protect, or a setting to turn on. Not being
  able to read the rule is treated exactly like the rule being off (Step 6,
  precondition 6).
- **A fix would touch `.github/workflows/` and the token cannot push there**
  — bucket that unit `deferred` with `gate: "error"` before editing anything
  (Step 4). This is a real degradation of the run, not a clean skip: the work
  is outstanding, it blocks auto-merge, and the report names it so a human
  picks it up. Do not retry it with a different credential.
- **A participant re-derivation refetch fails for a unit about to be acted
  on** — do not act on the assembly-time list. Bucket that unit `deferred`
  with `gate: "human"`, `classification_uncertain: true`, and the fetch error
  as the reason. An unrefreshed list is the stale evidence INV-3 exists to
  reject.
- **A third-party bot silent inside its window** — note it, continue.
- **No PR resolvable** — print a short note asking for a PR number or URL and
  stop. Do not guess a PR.
- **Push rejected / conflict** — stop touching that unit, bucket it
  `deferred` with `gate: "error"` and the exact error, continue with the rest.
- **HEAD moved between arming and the post-arm read** — not a degradation, a
  disarm: issue `--disable-auto`, report `armed: false` with both SHAs, and
  treat the new HEAD as new work. If `--disable-auto` itself fails, that is
  `INVARIANT_VIOLATION: INV-5` and the run stops — the PR is armed on evidence
  that no longer describes it, and no further automated action is safe.
- **A commit lands without the trailer** — this one does not degrade. Amend
  and re-verify; if the trailer still cannot be confirmed, do not push at all
  (INV-8). A push that the workflow's guard cannot recognise is how an
  unattended loop starts, so failing closed here is the cheap outcome.

## Dependencies

- **`pr-swarm`** (same plugin) — the review half, invoked with all four inputs:
  `Skill("pr-swarm", args="<pr> --head-sha <sha> --round <n>[ --dry-run]")`.
  Returns the JSON contract read in Step 2.
- `${CLAUDE_PLUGIN_ROOT}/skills/review-triage/references/triage-rubric.md` —
  classification rules, autonomy ladder, bot-login list, automated-signal
  signatures.
- `${CLAUDE_PLUGIN_ROOT}/skills/pr-swarm/references/github-ops.md` —
  exact `gh` invocations for the paginated thread fetch (§3), the per-thread
  comment walk past page 1 (§3), the paginated top-level issue-comment fetch
  (§2), replying, resolving, reading required checks and branch protection,
  and arming **and disarming** auto-merge (§9). Its thread fetch emits each
  participant as `{login, typename, automated_signal}` plus the per-thread
  `comments_walked` / `comments_has_next_page` — the exact field names INV-3's
  disposition rows carry, so they are consumed as they arrive and never
  renamed or recomputed here. Commands there marked unproven have been
  schema-introspected, not executed; a failure from one of those is a
  degradation to report, not a reason to improvise flags.
- `gh` CLI (`pr`, `api`, `repo`) and `git`. Locally both need
  `dangerouslyDisableSandbox: true`.

## Out of scope

Rebasing, changing the PR base, managing labels, batch mode across several
PRs, and reviewing non-PR working-tree diffs (that is `crosscheck:verify` from the local crosscheck plugin).
