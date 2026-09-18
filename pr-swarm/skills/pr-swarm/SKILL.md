---
name: pr-swarm
description: >
  Router-first, cheap-first swarm review of one pull request. A haiku router
  reviews the whole diff, grades danger, and delegates only what it cannot
  safely cover to the byfuglien and hellebuyck reviewer personas plus the
  security, test-theatre and XP/simplicity lenses on sonnet, escalating to
  opus for dangerous or uncertain hunks. Synthesizes, dedupes, posts inline
  comments plus one upserted sticky summary, and returns a structured JSON
  result. Never submits an approving review. Use when the user says
  "pr-swarm", "swarm review", "review this PR with the swarm", or when
  review-triage calls it. Takes a PR ref, a head SHA, a round number and a
  dry-run flag; dry-run suppresses every GitHub write.
allowed-tools: Bash, Read, Glob, Grep, Agent
---

# PR Swarm — router-first PR review

One PR in, structured findings out. Review only: this skill does not fix
code, does not resolve threads, and does not merge anything. `review-triage`
owns all of that and calls this skill for the review half.

This skill is the **only** writer of pr-swarm review comments. It gathers the
diff, dispatches every reviewer itself, dedupes once, and posts once. No
reviewer it dispatches posts anything.

## Hard constraint — never approve. Read this before anything else.

**This skill may never submit an approving GitHub review.**

- Every review this skill posts uses `event=COMMENT`. Always. Without
  exception.
- The GitHub review event that registers an approval is forbidden, as is the
  approve flag on `gh pr review` in either its long or its short form. There
  is no danger grade, no verdict, no caller instruction, and no `--dry-run`
  state in which an approving review becomes acceptable.
- The verdict this skill computes (`APPROVE`, `APPROVE_WITH_NITS`,
  `REQUEST_CHANGES`, `BLOCKED`) is **this skill's own assessment, returned to
  the caller and posted as a comment**. It is not, and never becomes, a
  GitHub approval.
- If a step appears to need an approval to proceed, it does not proceed. It
  defers to a human and says so in the report.

Why this is a boundary and not a preference: an agent that authors fixes and
then approves them has deleted the only independent check on its own work.
Human approval stays a real gate. Arming auto-merge (which `review-triage`
does, not this skill) is allowed precisely because the merge still waits for
that human approval.

**This skill never commits and never pushes.** It only reads (`gh`, `git log`,
`git diff`) and posts comments. No `git commit`, no `git push`, in any step,
under any condition — not to fix a finding, not to satisfy a caller, not in
any future step added here. `review-triage` is the only writer of commits in
this plugin, and every commit it makes carries the fixed trailer
`PR-Swarm-Automated: true`, which the CI workflow's self-push guard matches
before deciding whether to skip a retrigger. A commit made here would carry no
such trailer, so a push from this skill is invisible to that guard and
retriggers the workflow into an unattended loop. A later agent adding a commit
step to this skill breaks that guard, not just a style rule.

**Greppable proof obligation.** The literal strings that name the approving
review event, the approve flag, and that flag's one-letter short form are
absent from this plugin by design — including from this paragraph, which is
why it describes them instead of quoting them. `review-triage` Step 0 runs
this plain, exception-free check:

```bash
grep -rniE 'event=APPROV[E]|pr review[^;|&]*--approv[e]|pr review[^;|&]*[[:space:]]-[[:alnum:]]*[a][[:alnum:]]*([[:space:]]|$)' \
  "${CLAUDE_PLUGIN_ROOT}" .github/workflows 2>/dev/null
```

and treats **any** hit as an invariant violation that stops the run.

**This is one check with one spelling.** The bytes above are the bytes in
`review-triage` Step 0, which is where the check actually executes; this file
quotes them so a reader of either file sees the same command. Changing the
pattern is a coordinated edit to both files in one commit. Two spellings of a
safety check mean the one you read is not the one that ran.

Four things about that command are load-bearing:

- **No output is the pass.** `grep` exits **1** when it matches nothing, so
  the clean run is a non-zero exit with empty stdout. Do not read that exit
  code as a failed check. Exit **0** (something matched) is the violation.
  Exit **2** means part of the path list was unreadable, and `2>/dev/null`
  hides the diagnostic that would otherwise tell 2 apart from 1 — so exit 2
  with empty stdout looks exactly like the pass, and it is not one. Capture
  the status rather than inferring it from the output:
  `set +e; grep ...; rc=$?`.

  On `rc == 2`, establish the cause before deciding what it meant. Test
  whether `.github/workflows` exists. Absent — the run is outside the repo —
  means the plugin half really was checked: record a **degradation**, never a
  clean check over both halves. Present means something else in the path list
  was unreadable: the check is broken, so fix it and re-run rather than
  proceeding on it.
- **POSIX classes, not `\n` escapes.** Inside a bracket expression `\n` is not
  a newline; `[^\n]` means "not a backslash and not the letter n", so that
  spelling behaves differently on GNU grep (CI) and BSD grep (a laptop) —
  the worst possible property for a safety check. `[^;|&]` is the deliberate
  spelling: `grep` is line-oriented already, so the only thing worth
  excluding is a command separator, which stops `pr review` on one line from
  reaching an unrelated command after `;`, `|` or `&&`.
- **Both flag forms, bundles included.** The long flag and the one-letter
  short form of `gh pr review`'s approve flag are equally fatal, and a bundle
  like `-ab` approves just as effectively as the spelled-out flag — hence
  `-[[:alnum:]]*[a][[:alnum:]]*` rather than a bare `-a`. Both short-form and
  long-form branches are anchored to a preceding `pr review` on the same
  command, so an unrelated flag elsewhere in the plugin does not trip them.
- **The workflow is in scope.** The acceptance criterion names the plugin
  *and* the workflow, so the path list includes the repository's
  `.github/workflows` alongside `${CLAUDE_PLUGIN_ROOT}`. A ban enforced over
  half the surface it claims to cover is not a ban.

A hit in prose is indistinguishable from a hit in a command, and that is
deliberate: a dumb grep nobody can argue with beats a clever one with a
mention-versus-use exception, because the exception is what a later edit
widens. Never reintroduce any of those literals, in any file of this plugin
or of the workflow, for any reason — not in an example, not in a warning, not
in a test fixture.

## Bot identifier — REQUIRED on every posted comment

Every comment this skill posts — inline review comments, the review body,
the sticky summary, any thread reply — begins with:

```markdown
> [!NOTE]
> 🤖 Automated comment by **pr-swarm** — not written by a human
```

Apply the header at the outermost point where a body is assembled, so a body
built in pieces cannot ship without it. A comment without this header is a
bug, not a style slip. It is also load-bearing downstream: `review-triage`
identifies our own threads by this header, not by account type.

## Inputs

Four inputs, **all required** when another skill invokes this one:

| Input | Form | Meaning |
| --- | --- | --- |
| PR ref | `1870`, or `https://github.com/owner/repo/pull/1870` | Which PR to review. A bare number resolves the repo with `gh repo view --json nameWithOwner -q .nameWithOwner`; a URL supplies `owner`, `repo`, `number` directly. |
| Head SHA | 40-char SHA | The commit the caller believes is HEAD, and the anchor it expects the findings to be valid for. |
| Round | integer ≥ 1 | The caller's round number. Echoed in the return contract and in the sticky-summary header. |
| Dry-run | `--dry-run` present or absent | Present: do the entire review, write **zero** bytes to GitHub, and print the report plus the full rendered payload — every inline comment body and the whole sticky summary body, verbatim and unabridged (Step 6, *Under `--dry-run`*). Printing a count or a description of them instead is a failed dry run. |

`$ARGUMENTS` may carry these in any order, e.g.
`1870 --head-sha 3fd91ac... --round 2 --dry-run`.

**Defaults, and only for a direct human invocation** (`pr-swarm 1870`):
head SHA defaults to the `headRefOid` resolved in Step 1, round defaults to
`1`, dry-run defaults to off. A skill-to-skill invocation that omits any of
the four is a caller bug: proceed with these defaults but record
`missing input: <name> — defaulted` in `degradations[]` so the omission is
visible in the artifact rather than silently absorbed.

With no PR argument at all, detect the current branch's PR via
`gh pr view --json number,url`. If there is no PR, run the review against
`git diff <base>...HEAD` and report to the terminal only — posting is
skipped, not faked, and the return contract carries
`head_sha` = local `HEAD` with a degradation saying no PR was found.

Behaviour is otherwise identical in both invocation paths: same steps, same
caps, same output contract. Nothing in this skill branches on "who called me"
except the round number in the sticky-summary header and the defaulting rule
above.

### Sandbox note

Locally, every `gh` and `git` call in this skill needs
`dangerouslyDisableSandbox: true` on the Bash call — `gh` auth reads the
macOS Keychain and `git fetch`/`push` need network and credential access
that the default sandbox denies. In CI (`claude-code-action`) there is no
sandbox; run the same commands unchanged.

### Temp files — set `TMPDIR` before using it

Every shell block in this skill that writes a scratch file starts with:

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"
```

GitHub's Linux runners do **not** set `TMPDIR`; they set `RUNNER_TEMP`. macOS
sets `TMPDIR`, which is why this is invisible in local testing and fatal in
CI: with the variable unset, `"$TMPDIR/pr-swarm.diff"` expands to
`/pr-swarm.diff` and round 1 dies on permission denied at the first write.
Each block runs in its own shell, so the line is repeated per block rather
than set once — a block that uses `$TMPDIR` without it is the bug.

## Step 1 — Resolve the PR and gather the diff

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"

gh pr view "$PR" --repo "$OWNER/$REPO" \
  --json number,title,body,headRefName,headRefOid,baseRefName,author,url,files,additions,deletions,changedFiles,isDraft \
  > "$TMPDIR/pr-swarm-meta.json"

gh pr diff "$PR" --repo "$OWNER/$REPO" > "$TMPDIR/pr-swarm.diff"

gh api "repos/$OWNER/$REPO/pulls/$PR/comments" --paginate > "$TMPDIR/pr-swarm-existing-inline.json"
gh api "repos/$OWNER/$REPO/issues/$PR/comments" --paginate > "$TMPDIR/pr-swarm-existing-issue.json"
```

Both `--paginate` flags are load-bearing. An unpaginated fetch quietly hides
the 31st existing comment, and every dedupe decision downstream is then made
against a partial picture that still looks complete.

Record and carry forward:

- `HEAD_SHA` = `headRefOid` **as freshly read here**. This is the SHA the
  review is performed against, the SHA every inline comment anchors to, and
  the SHA returned as `head_sha`. If it differs from the head SHA the caller
  passed, the branch moved between the caller's read and ours: review the
  fresh SHA, return the fresh SHA, and add
  `head sha moved: caller <passed> → reviewed <HEAD_SHA>` to
  `degradations[]`. Never return the caller's SHA for a review performed
  against a different tree — that is precisely the check `review-triage` uses
  the field for.
- `SHORT_SHA` = first 7 chars, for the summary header.
- Changed-file list, total added+deleted lines, commit log
  (`git log <base>...<head> --oneline`).
- **The automated-comment index** — see below. Step 5 dedupes against it.

### The automated-comment index

From the two fetched comment files, build the set of
`(path, line, source, concern)` tuples already carrying a comment from **any
automated reviewer**, not just from pr-swarm. A comment is automated when any
of these holds:

- its author is a GitHub App / bot — `user.type == "Bot"`, or a login ending
  in `[bot]` (covers Copilot, Claude, Dependabot, CodeRabbit and friends);
- its body carries the pr-swarm bot-identifier header (our own prior rounds —
  our comments post through a human-typed account, so the header is the only
  reliable marker);
- its body carries the Crosscheck signature `Authored by Crosscheck` — a PR
  may already carry comments from an earlier manual `crosscheck-pr-review`
  run, and those cover the same ground our byfuglien/hellebuyck pass does.

Anything else is a human comment and is **not** a dedupe source — a human
raising a point does not mean an automated reviewer has covered it.

**A top-level automated comment has no `(path, line)`, and is a dedupe source
anyway.** Index it as `(null, null, source, concern)` and dedupe unanchored
findings against it. The tuple shape is a convenience for inline threads, not
the definition of coverage: `crosscheck-pr-review` posts its consolidated
notes as top-level issue comments, and those are exactly the general,
file-level concerns this skill's own unanchored findings collide with.
Dropping them because they carry no anchor is how a review re-raises, under
its own signature, a point an earlier automated reviewer already made on the
same PR.

**Classify by the signals above, never by author type or by eye.** The
Crosscheck signature and the pr-swarm header both sit in bodies posted from a
human-typed account (`user.type == "User"`), so a comment that *looks* human
in the author column is routinely automated — that is the entire reason the
body-signature rules exist. Run `github-ops.md` §2's `automated_signal` jq
over the fetched top-level comments and read its verdict; do not form a
second opinion from the author list. A report may state that a PR carries no
existing automated comments **only** when that jq returned `none` for every
one of them; saying so while comments classify `header:crosscheck` is a
false premise for every dedupe decision downstream, and it is reported as a
defect of the run, not as a clean PR.

**Two `claude[bot]` comments on this PR are not ours, and neither is a dedupe
source.** In CI the action runs in tag mode, which posts its own top-level
progress-tracking comment on every run — a checklist and a job link, written
by the runner, not by this skill. It is authored by a bot, so the first rule
above would otherwise adopt it. Exclude it — but on what it *says*, not on its
missing anchor: it is a progress checklist and a job link, carrying no
finding and therefore covering no concern. Do not generalise the exclusion
to "no `(path, line)` means no tuple"; a Crosscheck top-level comment also
has no anchor and does cover concerns (see above). It is also **not** the sticky summary — the summary is found by its
`<!-- pr-swarm-summary -->` marker, never by bot identity, which is the same
reason the workflow sets `use_sticky_comment: false`. Treating the tracking
comment as a prior pr-swarm comment would make this skill dedupe its own
findings against a progress checklist.

## Step 2 — Load the reviewer bodies

Read `${CLAUDE_PLUGIN_ROOT}/skills/pr-swarm/references/lenses.md`. One read,
one file: it holds the self-contained prompt body for **every** reviewer this
skill can dispatch. Three extra lenses —

| Lens tag | Covers |
| --- | --- |
| `security` | authn/authz, IDOR, injection, SSRF, secrets, prompt injection |
| `test-theatre` | tests that assert nothing meaningful; missing coverage of the changed behaviour |
| `xp` | simplicity, duplication, naming, cohesion, YAGNI, over-abstraction |

The same file also holds the **crosscheck pair** — the byfuglien and
hellebuyck reviewer personas, vendored into this plugin (with attribution)
from the local `crosscheck` plugin in this marketplace:

| Reviewer tag | Lens |
| --- | --- |
| `crosscheck/byfuglien` | correctness, invariants, fault paths, patch equivalence, verification adequacy |
| `crosscheck/hellebuyck` | spec coverage, intent alignment, governance scaffolding, protected surfaces, invariant coverage |

**Read both persona bodies from `lenses.md`, from the section carrying the
reviewer tag above — nowhere else.** In particular, do not read
`~/.claude/skills/crosscheck-pr-review/SKILL.md` or any other path under
`~/.claude`: that is one developer's user-level install, absent in CI and on
every other machine, so a skill that depends on it works exactly where it was
written and silently loses its two strongest lenses everywhere else — which,
on a mandatory-trigger diff, means a `HIGH` finding for "required lens
unavailable" on every deploy-path PR.

For the same reason, **do not dispatch these two as `crosscheck:byfuglien` /
`crosscheck:hellebuyck` subagent types.** Those agents exist only where the
`crosscheck` plugin is installed. Dispatch each as a plain `Agent` with the
default subagent type (`general-purpose`) carrying the vendored body as its
prompt, the same way the three lenses above are dispatched. One dispatch
path, identical locally and in CI; this plugin declares no dependency on the
`crosscheck` plugin and must not grow one.

**Do not invoke `crosscheck-pr-review` as a skill either.** That skill is a
complete review pipeline: it posts its own inline comments and its own
summary the moment it runs. Calling it here would break `--dry-run` (it
writes regardless of our flag), double-post findings this skill also posts,
put a second signature and a second dedupe path on the PR, and hand back a
prose report where per-finding structure is needed. This skill owns every
GitHub write in the plugin; the personas are dispatched as reviewers only,
and are told explicitly to post nothing.

Availability check: every lens and both personas are usable exactly when
their body is present in `lenses.md`, so the single read above is the whole
check. A body that is missing or empty marks that lens unavailable now —
Step 3 routes around it (see *Graceful degradation*).

## Dispatch mode — every `Agent` call in this skill sets `run_in_background: false` where the harness offers it

Read this before Step 3. It governs **every** `Agent` dispatch this skill
makes: the router pass, every delegation, both crosscheck personas, and any
re-delegation.

```
Agent(
  subagent_type: "general-purpose",
  model: "haiku",            # or sonnet / opus, per the model policy
  run_in_background: false,  # REQUIRED wherever offered — see below
  description: "...",
  prompt: "..."
)
```

**The parameter is `run_in_background`, it is a boolean, and its default is
`true`.** That is the harness default, not this skill's choice: Claude Code's
own `Agent` tool documentation reads "Agents run in the background by
default… Pass `run_in_background: false` only when your very next action
depends on the agent's result and nothing else could usefully happen while it
runs." Every dispatch here is exactly that case — Step 5 cannot synthesize
anything until the reviewers have returned — so every dispatch passes
`false`, explicitly, on every call. Some harness configurations do not offer
the parameter at all; that case is **not** a licence to assume synchrony, and
is handled below.

Why it is spelled out rather than left to the default: on PR #24 this skill
dispatched four lenses without the parameter, got four launch receipts back
instead of four reviews, and spent **25 minutes of wall clock** waking on
completion notifications to reassemble them — against the 4.6 minutes it
reported for itself. The cost and latency reasoning everywhere below (the cap
of 6, the router-first ordering, "dispatch independent delegations in
parallel") assumes **parallel-and-awaited** dispatch: N agents launched
together in one message and all N results in hand when that message's tool
results land. Background dispatch is the opposite shape, and inheriting it
silently turns one parallel round into N serial wakeups.

Parallel and awaited are separate knobs and you need both:

- **Parallel** comes from putting all of a round's `Agent` calls in **one
  message, as multiple tool-use blocks**. One call per message is serial no
  matter what `run_in_background` says.
- **Awaited** comes from `run_in_background: false` on each of those calls.

Never pass `true`, and never leave the flag unset where it is offered
because the default looks convenient: an unset flag there is a 5× latency
regression that reports itself as fast.

### Where the parameter is absent, do not infer that dispatch is synchronous

`run_in_background` is dropped from the `Agent` schema for two **opposite**
reasons, and only one of them leaves the calls awaited. Verified in Claude
Code 2.1.274, which omits the parameter when background tasks are disabled
**or** when the fork gate is on:

- **Background tasks disabled.** Subagents are synchronous. `Bash` loses its
  own `run_in_background` at the same time. Pass nothing and carry on — the
  calls are already awaited.
- **Fork gate on.** Subagents still run in the background, `Bash` *keeps* its
  `run_in_background`, and there is **no flag to await them with**. Every
  dispatch returns a launch receipt, and each result arrives as a completion
  notification in a later turn.

Decide which one you are in by reading the `Agent` tool's own description in
this session, not by reasoning from the schema or from this paragraph's
version number: a description saying **only synchronous subagents** are
supported is the first case; one saying **subagents run in the background**
while offering no flag is the second. That is a live check, so it survives a
harness that moves either gate.

In the second case the parallel-and-awaited cost model below **does not
hold**, and the run must not report as if it did. Still put every `Agent`
call of a round in one message — parallelism is unaffected, only awaiting is.
Then wait for the completion notifications: do not wake-poll, and do not
begin Step 5 on partial returns. Record it as a degradation, in
`degradations[]` and in the report, e.g.

```
dispatch backgrounded — run_in_background not offered by this harness;
round latency is N completion notifications, not one parallel round
```

so that a 25-minute round is never again reported as a 4.6-minute one.

## Step 3 — Router pass (haiku)

Dispatch **one** `Agent` pinned to **`haiku`**, with
`run_in_background: false` (see *Dispatch mode* above — this pass gates
everything after it, so it is awaited, never backgrounded). This is the entry
reviewer and the cheap default. Pass it the full diff, the changed-file list,
and the commit log.

Tell it, verbatim in substance:

- You are the sole first-pass reviewer. Review the whole diff for
  correctness, bugs, security and style. Before judging any hunk, read at
  least 50 lines of surrounding context in the file.
- Grade the change's blast radius: `LOW` / `MEDIUM` / `HIGH` / `CRITICAL`,
  using the danger rubric below.
- Decide what to delegate to a stronger model, and to which lens, and over
  which scope. Delegate only what your own pass cannot safely cover, scoped
  to the specific files or hunks that worry you. **Delegating nothing is a
  legitimate, expected outcome** on a small low-danger diff — an empty plan
  is the cheap path working, so do not pad it.
- Post nothing to GitHub. Return findings; the orchestrator posts.
- Return your own findings in `STRUCTURED_FINDINGS` form, then
  `OVERALL_SUMMARY`, then a `DELEGATION_PLAN` block. All three, always.

### Danger rubric

Any of these raises blast radius and pushes toward delegation:

1. Touches auth, authorization, sessions, secrets or crypto.
2. Touches migrations, schema, ingestion or destructive DB writes.
3. Touches concurrency, locking or shared mutable state.
4. Touches billing, payments or anything with direct revenue impact.
5. Diff is large (>~400 lines) or spans many files with non-obvious
   cross-file effects.
6. Uses patterns the router is unsure about (unfamiliar framework, subtle
   async, off-by-one-prone loops).
7. The router found a HIGH/CRITICAL finding it wants confirmed.

### MANDATORY DELEGATION — fires regardless of the danger grade

At least one delegation is **required**, whatever grade the router assigned
and however confident it is, when the diff touches any of:

- **more than ~400 changed lines**
- **auth or permissions**
- **billing or payments**
- **database migrations or schema changes**
- **concurrency, locking, or shared mutable state**
- **deploy, release, or CI/CD paths**

A `LOW` grade does not excuse this, because **the grade is the thing being
checked**. A router has graded an 806-line rewrite of a production deploy
workflow `LOW` with high confidence and no delegation, on a diff that
carried a HIGH defect it missed. The trigger list is objective and
mechanical on purpose: check it against the changed-file list yourself, in
the orchestrator, *after* the router returns. If the router's plan is empty
and any trigger matched, **you add the delegation** — scope it to the
triggering files, route it to the lens that covers the trigger (auth/billing
→ `security`; migrations/concurrency → `crosscheck/byfuglien`;
deploy/release → `crosscheck/byfuglien`; size → the crosscheck pair), and
note the override in the report.

### Delegation plan format

The router returns this immediately after `OVERALL_SUMMARY`:

```
DELEGATION_PLAN:
danger: <LOW|MEDIUM|HIGH|CRITICAL>
confidence: <HIGH|MEDIUM|LOW>
delegations:
- model: <sonnet|opus> | lens: <crosscheck/byfuglien|crosscheck/hellebuyck|security|test-theatre|xp|general> | scope: <file paths / hunks / "full"> | reason: <one line>
...
(empty list if no delegation)
```

A router entry of `lens: crosscheck` (unqualified) expands to **both**
personas.

If `delegations` is empty **and** no mandatory trigger matched, skip Step 4 —
the router's findings are the review.

## Step 4 — Delegation pass

Only runs when the plan (router's, plus any mandatory override) is non-empty.

**Model policy — which model runs which pass:**

| Pass | Model | When |
| --- | --- | --- |
| Router (Step 3) | `haiku` | Always. Every run starts here. |
| Delegation (Step 4) | `sonnet` | Default for every delegation. |
| Delegation (Step 4) | `opus` | Only for: a hunk the router graded `HIGH`/`CRITICAL`; a router `confidence: LOW`; a mandatory-trigger scope the sonnet pass returned uncertain on; or two lenses disagreeing on the same hunk. |

Never pin `opus` because the diff is merely long. Length alone routes to
`sonnet`. Escalate on danger and uncertainty, not size.

**Cap: 6 delegations per pr-swarm run**, counting every dispatched agent.
`crosscheck/byfuglien` and `crosscheck/hellebuyck` are two separate agents
and therefore cost **two** from the cap; re-delegations cost one each.
Running low on budget: spend it on mandatory-trigger scopes first, then
`HIGH`/`CRITICAL` hunks, then everything else. When the cap truncates the
plan, say so in the report, in the sticky summary, and in `degradations[]` —
a truncated review is reported as truncated.

Dispatch independent delegations **in parallel and awaited**: one message
carrying every `Agent` call of the round as a separate tool-use block, and
`run_in_background: false` on each of them (*Dispatch mode*, above — both
knobs, every call, the crosscheck personas included). A round dispatched one
call per message is serial; a round dispatched without the flag comes back as
launch receipts and is reassembled over N wakeups. Each delegation:

- gets the diff **scoped to its entry's `scope`**, not the whole PR diff;
- is told it is the **sole reviewer** for that scope — no lens is told about
  the router or the other lenses, so convergence in Step 5 means something;
- is told to **post nothing to GitHub**; it returns findings, and this skill
  posts them;
- gets its prompt body verbatim from `lenses.md` — the three lenses and the
  two vendored crosscheck personas alike — wrapped in a self-contained prompt
  carrying the PR number, repo, title, the scoped diff and `HEAD_SHA`, since
  a subagent inherits none of this skill's context;
- returns `STRUCTURED_FINDINGS` + `OVERALL_SUMMARY` in the shared format
  below. **Require that format from the crosscheck personas too** — ask for
  it explicitly in their prompt. If a persona answers in the crosscheck
  severity vocabulary anyway, map it: `blocker→CRITICAL`, `major→HIGH`,
  `minor→MEDIUM`, `nit→NIT`, and set `reviewer` to `crosscheck/byfuglien` or
  `crosscheck/hellebuyck`. A persona that returns prose with no parseable
  findings block counts as a degradation, not as a clean review: record it in
  `degradations[]`.

A second delegation to a different lens is allowed when a first delegation's
findings point at a lens that did not run — it just spends from the same cap
of 6.

### Shared output format — STRUCTURED_FINDINGS

Every reviewer, router included, ends its response with exactly this:

```
STRUCTURED_FINDINGS:
[
  {
    "file": "<path>",
    "line": <number or "general">,
    "severity": "<CRITICAL|HIGH|MEDIUM|LOW|NIT>",
    "reviewer": "<tag>",
    "confidence": "<HIGH|MEDIUM|LOW>",
    "body": "<the review comment text — may be several paragraphs, may contain markdown and fenced code>"
  }
]

OVERALL_SUMMARY:
<one paragraph assessment>
```

With nothing to report:

```
STRUCTURED_FINDINGS:
[]

OVERALL_SUMMARY:
<one paragraph assessment>
```

**A JSON array, one object per finding — not one pipe-delimited line per
finding.** An earlier version of this format specified a single
`|`-separated line ending in `body: <text>`, and every lens in the first
five-lens run silently returned JSON instead. They were right to: a review
body is multi-paragraph markdown, routinely containing newlines, pipes in
tables and fenced code, none of which a one-line pipe-delimited record can
carry. The format was unwritable for the content it demanded, so 5/5
non-compliance was a defect in the format, not in the lenses.

`severity` is exactly one of the five values above. **`INFO` is not a
severity.** A lens wanting to record what it checked and found clean —
mutation results, controls it verified, ground it cleared — puts that in
`OVERALL_SUMMARY`, where Step 5 reads it as evidence of coverage. A finding
is something a reader should act on; everything else is the summary's job.

`reviewer` tags: `router`, `crosscheck/byfuglien`, `crosscheck/hellebuyck`,
`security`, `security/<category>` (e.g. `security/idor`), `test-theatre`,
`xp`.

## Step 5 — Synthesize

1. **Dedupe across reviewers.** Findings on the same file within 5 lines, or
   clearly the same concern, merge into one. Keep the **highest** severity,
   concatenate the bodies, and tag the merged finding
   `[convergent: <lens-a> + <lens-b>]`. Convergence is the strongest signal
   in the run — two independent lenses hit the same spot — so say so in the
   comment and raise confidence accordingly.
2. **Dedupe against what is already on the PR.** Drop a finding when the
   automated-comment index from Step 1 already carries **the same concern**
   at the same `(path, line)` — whatever automated source posted it: a prior
   pr-swarm round, a Crosscheck-signed comment from a manual run, or another
   review bot. Record each such drop as
   `deduped against <source> at <path>:<line>` in the report. Two different
   concerns that happen to land on the same line are not duplicates: keep
   ours. Human comments never suppress a finding.
3. **Drop** `NIT` findings with `confidence: LOW`.
4. **Keep** every `CRITICAL` and `HIGH` finding unconditionally.
5. **Demote, never drop, an unanchorable finding.** A finding whose
   `(path, line)` is not inside a diff hunk — including every
   `line: general` finding — cannot become a review thread. Route its text to
   the sticky summary **and** return it in `findings[]` with
   `anchored: false` and its full text in `body`. This matters:
   `review-triage` triages threads, and an unanchored finding never appears
   in a thread fetch, so a finding that goes only to the summary escapes
   triage entirely. The `anchored` flag is what pulls it back into triage,
   and `body` is the only copy of its text `review-triage` will ever see —
   the summary is prose it does not parse. An unanchored finding returned
   without its text cannot be classified at all.
6. **Order** by severity, then path, then line.

**Verdict:**

- Any `CRITICAL` → overall risk `CRITICAL`
- 2+ `HIGH`, or 1 `HIGH` + 2 `MEDIUM` → `HIGH`
- 1 `HIGH`, or 3+ `MEDIUM` → `MEDIUM`
- Only `LOW`/`NIT`/none → `LOW`

Mapped to the returned verdict:

| Risk | `verdict` (contract value) | Displayed as |
| --- | --- | --- |
| LOW | `APPROVE` | ✅ **APPROVE** |
| MEDIUM | `APPROVE_WITH_NITS` | 💬 **APPROVE WITH NITS** |
| HIGH | `REQUEST_CHANGES` | ⚠️ **REQUEST CHANGES** |
| CRITICAL | `BLOCKED` | 🚫 **BLOCKED** |

Return the middle column's value exactly, underscores included. The displayed
form is for human eyes in the sticky summary only.

`APPROVE` here is a *word in a comment and a value in a JSON field*. It is
never a GitHub approving review, and the skill never submits one. See the
hard constraint at the top.

## Step 6 — Post to GitHub

### A dry run leaves no record, and the next round cannot see it

Round N+1 reconstructs prior rounds from what is **on the PR** — the sticky
summary and the existing inline comments. A dry run posts none of that, so
its findings survive only in the terminal. Two consequences, both to be
stated in the report rather than discovered later:

- Consecutive dry runs on the same head are **independent samples, not
  rounds**. They will not agree finding-for-finding, and neither can dedupe
  against the other. Two dry runs of this skill against `#24` differed by
  four findings and by a whole verdict tier, because one of them found an
  invariant-ID collision with `main` and the other did not.
- A dry run that found a CRITICAL has recorded it nowhere. Either post the
  round for real, or carry the finding out of the terminal by hand, before
  running again on the same head — a rerun is not a re-derivation, and
  severity does not survive a coin flip.

### Under `--dry-run`: print the payload, post nothing

`--dry-run` suppresses the writes in 6a and 6b; it does **not** suppress the
step. Render everything this step would have sent, print it, then go to
Step 7. Zero bytes to GitHub.

**Printing the payload is required, not a courtesy.** A dry run whose output
is a summary of what it would post ("would post 3 inline comments and update
the sticky summary") is a failed dry run and is reported as one: reviewing
the exact words before a live run is most of what the flag is for, and those
words cannot be reviewed from a count. This has been got wrong in a real run
— the review happened, the bodies never appeared, and nobody could say what
the live run would have said.

Print, in this order, in fenced blocks, verbatim:

1. **Every inline comment body from 6a — one block per finding**, in the
   order they would be posted, each preceded by its
   `<path>:<line>` anchor and its `anchored` value. The block holds the
   **fully rendered body**: the `> [!NOTE]` bot-identifier header, the
   `**[<tag>]** <emoji> <SEVERITY>` line and the finding prose, exactly as
   6a's body shape composes them. With no inline comments to post, print the
   single line `0 inline comments would be posted` — an empty section is
   ambiguous, and silence reads as a forgotten step.
2. **The whole sticky summary body from 6b — one block**, from the
   `<!-- pr-swarm-summary -->` marker through the closing
   `*Automated by pr-swarm — not a human review*` line, with every section
   filled in as it would be written: verdict header, key findings,
   unanchored findings, convergence, the coverage table, and the collapsed
   previous-rounds block. Print it whether it would have been a create or a
   PATCH, and say which it would have been, with the comment id it would
   have PATCHed.
3. **The exact commands**, each marked as not executed: the
   `POST .../pulls/{pr}/reviews` (or the individual-comment POSTs on the
   fallback path), the summary lookup, and the `PATCH`/`gh pr comment` that
   would follow — same flags, same paths, same `HEAD_SHA` as a live run.

Nothing is elided: no ellipses, no truncation, no "as above", no "(body
omitted for brevity)". The printed bytes are the deliverable. Long output is
the expected shape of a dry run on a large review, and shortening it defeats
the flag.

`findings[]` in Step 7 is unaffected by the flag except for one field:
`anchored` still reports truthfully (the in-hunk test is computed either
way) and `body` is still present in full; only `comment_id` is absent for
every entry, because nothing was posted to have an id.

### 6a — Inline comments

Post as a single review with **`event=COMMENT`**, anchored at `HEAD_SHA`;
fall back to individual `pulls/{pr}/comments` POSTs if the batched review
body gets unwieldy. Exact commands: `references/github-ops.md`.

Capture the review-comment id for each posted finding — it becomes
`comment_id` in the return contract, and it is what lets `review-triage`
match a thread back to the finding that created it.

Getting it takes one extra read. The batched `POST .../pulls/{pr}/reviews`
response is the **review** object: it carries the review's own id, not an id
per comment. Take that id, then list the PR's review comments and keep the
ones the review created:

```bash
TMPDIR="${TMPDIR:-${RUNNER_TEMP:-/tmp}}"

# redirect the POST in github-ops.md into this file; by default it prints
# the response and drops it
REVIEW_ID=$(jq -r '.id' "$TMPDIR/pr-swarm-review-response.json")

gh api "repos/$OWNER/$REPO/pulls/$PR/comments" --paginate \
  --jq "[.[] | select(.pull_request_review_id == $REVIEW_ID)
        | {id, path, line, start_line, body}]"
```

Map each row back to its finding by `(path, line)` — the pair is unique
within a round, because Step 5 already merged findings that shared an anchor.
The `id` from that row is the `comment_id` value. Taking the individual-POST
fallback path instead, read `.id` straight off each POST response; it is the
same id space.

`--paginate` again: a review with more than 30 comments spills onto a second
page, and without it the findings past the first page silently return with no
`comment_id` and get re-classified from scratch next round.

Body shape:

```markdown
> [!NOTE]
> 🤖 Automated comment by **pr-swarm** — not written by a human

**[<reviewer_tag>]** <severity emoji> <SEVERITY>

<finding body>
```

Severity emojis: 🔴 CRITICAL, 🟠 HIGH, 🟡 MEDIUM, 🟢 LOW, ⚪ NIT.
Convergent findings use `**[convergent: <a> + <b>]**` in place of the single
tag.

A 422 on a POST means the anchor is not in the diff. Demote that finding to
the sticky summary, flip its `anchored` to `false`, and carry on; do not
retry the same anchor.

### 6b — Sticky summary (upsert, exactly one per PR)

pr-swarm maintains **one** top-level comment per PR, marked with the HTML
comment `<!-- pr-swarm-summary -->`. Re-runs **PATCH that comment in place**.
A second summary comment on the same PR is a defect.

**This skill is the sole writer of that comment, and it regenerates the whole
body every round.** Nothing else may write into it — not `review-triage`, not
the workflow. There is no append path and no reserved section: anything
another writer added would be overwritten by the next round's regeneration
without warning, so a note placed there is not a record. `review-triage`
records its own outcomes in its run report, which is its record.

Find it first:

```bash
gh api "repos/$OWNER/$REPO/issues/$PR/comments" --paginate \
  --jq '[.[] | select(.body | contains("<!-- pr-swarm-summary -->"))][0].id'
```

Found → `PATCH repos/$OWNER/$REPO/issues/comments/<id>`. Not found → create
once with `gh pr comment`.

Body shape:

```markdown
<!-- pr-swarm-summary -->
> [!NOTE]
> 🤖 Automated comment by **pr-swarm** — not written by a human
>
> Router-first swarm review. This is a comment, not an approval.

## Verdict: <emoji> <VERDICT> <sub>(round <N> @ <SHORT_SHA>)</sub>

<1–2 sentences: why this verdict>

### Key findings
<top findings, grouped by severity — current round only>

<Any severity tally written here (`🟢 LOW ×4`, "2 MEDIUM", and the like) is
**counted from `findings[]`**, over every entry of that severity, anchored and
unanchored alike — not from the subset named in this section's prose. A
summary that says `LOW ×4` above an unanchored list holding a fifth LOW
contradicts the contract the same run returns, and the reader has no way to
tell which number is the real one. Either tally them all or name no count.>

### Unanchored findings
<every finding returned with `anchored: false`, as
`<path>:<line-or-general> — <severity> — <one line>`; omit this section when
there are none>

### Convergence
<findings two or more lenses hit independently>

### Coverage
| Lens | Ran? | Assessment |
| --- | --- | --- |
| 🧭 router (haiku) | yes | <danger grade, confidence, what it delegated> |
<one row per lens that actually ran; omit lenses never dispatched; list
unavailable lenses with "skipped — <reason>">

<details>
<summary>Previous rounds (<n>)</summary>

<one compact line per prior round: `round <N> @ <sha> — <verdict>: <one-line disposition>`,
derived from the existing comment's current header plus its own history block>

</details>

---
*Automated by pr-swarm — not a human review*
```

`<N>` in the header is the **round input**, echoed verbatim — not a counter
this skill maintains. The current round's detail collapses to a single
history line on the next update; it is not carried verbatim, so the comment
does not grow without bound.

## Step 7 — Return

Two outputs, both required.

### 7a — The return contract (binding)

Return this JSON object. `review-triage` consumes exactly these fields, and a
missing one is a contract break, not a formatting nit:

```json
{
  "head_sha": "3fd91ac0f1c2b4e6a8d90571e3b2c4a6d8091f2e",
  "round": 2,
  "verdict": "APPROVE_WITH_NITS",
  "router": { "danger": "MEDIUM", "confidence": "HIGH", "delegations_issued": 3 },
  "findings": [
    { "file": "src/app.py", "line": 42, "severity": "HIGH", "reviewer": "crosscheck/byfuglien", "body": "`_resolve_user` trusts the `user_id` query param without checking it against the session subject, so any authenticated caller can read another user's record. Look it up from the session instead.", "anchored": true, "comment_id": 2317745501 },
    { "file": "src/app.py", "line": "general", "severity": "MEDIUM", "reviewer": "xp", "body": "Three near-identical retry wrappers were added in this PR (lines 88, 140, 206). Collapse them into one helper before a fourth appears.", "anchored": false }
  ],
  "degradations": ["crosscheck/hellebuyck unavailable — persona body missing from lenses.md"]
}
```

Field rules:

- `head_sha` — the SHA this review was actually performed against (Step 1),
  never the SHA the caller passed if the two differ. `review-triage` rejects
  a verdict whose `head_sha` is not current HEAD, and it cannot evaluate its
  auto-merge preconditions without this field.
- `round` — the caller's round number, echoed unchanged.
- `verdict` — exactly one of `APPROVE`, `APPROVE_WITH_NITS`,
  `REQUEST_CHANGES`, `BLOCKED`. An assessment, never a GitHub review event.
- `router` — `danger` (`LOW|MEDIUM|HIGH|CRITICAL`), `confidence`
  (`HIGH|MEDIUM|LOW`), `delegations_issued` (integer, counting every agent
  dispatched in Step 4, mandatory overrides included).
- `findings[]` — one entry per finding that survived Step 5, posted or not.
  `file`; `line` (integer, or `"general"`); `severity`; `reviewer` (the tags
  from Step 4); `body`; `anchored`; `comment_id` (optional). The last three
  carry the weight:

  - **`body` — required on every entry, and never empty.** The finding's own
    review text: the same prose that went into the inline comment or the
    sticky summary, minus the bot-identifier header and the
    `**[<tag>]** <emoji> <SEVERITY>` line, which are presentation. For a
    convergent finding it is the merged text. `review-triage` classifies a
    finding as actionable, nit or ambiguous **entirely from this prose** —
    that is what its rubric reads. An anchored finding it can re-read from
    the thread; an `anchored: false` finding exists nowhere else, so an entry
    without `body` is not "a bit thin", it is unclassifiable, and the whole
    unanchored-triage path dies with it. Keep it as written rather than
    summarising it: a classifier reading a summary of a finding is not
    reading the finding.
  - **`anchored`** (boolean) — `true` only for a finding that became, or in
    dry-run would become, an inline review thread. Every `anchored: false`
    entry is a finding `review-triage` must triage from this list, because it
    exists nowhere in a thread fetch.
  - **`comment_id`** (integer) — present exactly when `anchored` is `true`
    **and** the comment was really posted this run; absent otherwise, and
    absent for the whole list under `--dry-run`. Its value is the
    **pull-request review comment id** of the thread's root comment — the
    `id` under `repos/{owner}/{repo}/pulls/comments/{id}`, which is the same
    integer `github-ops.md`'s thread walk returns as `first_comment_id`
    (GraphQL `comments.nodes[0].databaseId`). **Its one purpose is mapping a
    thread back to the finding that created it**: a thread unit whose
    `first_comment_id` equals a `comment_id` in this list is this round's own
    thread, so `review-triage` triages it from the finding beside it — known
    severity, known body, known reviewer — instead of re-reading it as if a
    stranger had written it. That is the only consumption; do not describe it
    as deduping `findings[]`. It cannot dedupe an unanchored finding, because
    `anchored: false` and a `comment_id` are mutually exclusive by the rule
    two lines up, and unanchored findings are the only ones that enter
    `review-triage` as finding units. It is not an issue-comment id and not a
    review id; those live in different id spaces and comparing across them
    matches nothing while looking like it works.
- `degradations[]` — every lens, persona, model or signal that was
  unavailable or downgraded this round, one string each. Empty array when
  nothing degraded; never omit the key.

All six top-level keys are always present. Empty arrays are fine; absent keys
are not. Within `findings[]`, `file`, `line`, `severity`, `reviewer`, `body`
and `anchored` are always present; only `comment_id` is optional.

### 7b — The human report

Also print a report under ~400 words:

1. **Verdict** and the risk tier behind it.
2. **Head SHA reviewed**, and whether it matched the caller's.
3. **Router grade** — danger, confidence, what it delegated and why.
4. **Delegations run** — lens, model, scope, one-line outcome. Flag any
   mandatory-trigger override you added, and any truncation at the cap of 6.
5. **Findings** — count by severity; convergent ones marked; unanchored ones
   listed separately.
6. **Dropped** — deduped-against-existing (with source), low-confidence nits.
7. **Degradations** — the same list as the contract field, in prose.

## Graceful degradation — warn and downgrade, never abort

| Failure | Behaviour |
| --- | --- |
| `haiku` unavailable / rejected by the harness | Run the router unpinned (session model). Warn. The pass still produces a grade and a plan. |
| `sonnet` unavailable | Delegate on the session model; note the substitution in the report and in `degradations[]`. |
| `opus` unavailable | Run the escalation on `sonnet` and mark the finding `confidence: MEDIUM` at best, with a line in the report saying the escalation was downgraded. |
| A crosscheck persona's body missing or empty in `lenses.md` | Skip that persona. Warn. Re-route its scope to the other persona, or to `security` + `xp`, or cover it in the router pass. If a mandatory trigger routed there and nothing can cover it, file a `HIGH` finding recording that the required lens was unavailable. This is a packaging defect (the body ships inside the plugin), not an environment difference — say so in the degradation line. |
| A crosscheck persona returns prose with no parseable findings block | Treat it as no coverage, not as a clean bill: record the degradation and re-route its scope if budget allows. |
| A lens body missing from `lenses.md` | Skip that lens, warn, re-route if another lens fits. |
| No PR detected | Review the branch diff, report to the terminal, post nothing; return the contract with the local HEAD and a degradation. |
| GitHub write fails (auth, rate limit) | Keep the findings, report them in full to the terminal, return them in `findings[]` with `anchored: false`, and state clearly that posting failed. Never silently drop findings. |
| `run_in_background` not offered and subagents still run in the background (fork gate) | Dispatch the round in parallel anyway, await the completion notifications, and record the latency shape in `degradations[]` and the report — see *Dispatch mode*. The review is unaffected; only its cost model is. |
| Only the router available | Still post its findings. A cheap pass is a real review; say in the summary that the swarm was degraded to router-only. |

Every row writes a line into `degradations[]`. Nothing in this table aborts
the run, and nothing in it relaxes the hard constraint at the top.
