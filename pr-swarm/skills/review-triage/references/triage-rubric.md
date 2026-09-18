# review-triage — classification rubric

Rules `SKILL.md` Step 4 delegates to: the human-participation gate, the
actionable/nit/ambiguous classification, the autonomy ladder for ambiguous
bot threads, and the bot-login list used by both the classification and the
`pr-swarm` sub-tag mapping. `SKILL.md` owns control flow (order of
operations, the bucket counts, the invariants); this file owns the judgement
calls inside each step.

## 1. Human-participation gate — decide this before anything else

A thread is **human** if **any** comment in it — not just the first — is not
provably automated. Provably automated means **either**:

- the comment's author has GraphQL `__typename == "Bot"`, or a login ending
  in `[bot]`, **or**
- the login is on the bot list below, **or**
- the comment body carries the pr-swarm bot-identifier header
  (`🤖 Automated comment by **pr-swarm**` — see `SKILL.md`'s bot-identifier
  section), or the `_🛡️ Authored by Crosscheck 🛡️_` signature (vendored,
  with attribution, into this plugin's own
  `../../pr-swarm/references/lenses.md` — this plugin does not depend on the
  external `crosscheck-pr-review` skill for this signature, or for anything
  else), or the header from any other known reviewer skill in the bot list.

This is deliberately body-content-first, because the account type our own
comments post through is not the same in both places this skill runs.
**Locally**, `pr-swarm`, `review-triage`, and a manually-run
`crosscheck-pr-review` all post through the invoking user's own
GitHub-authenticated account (`__typename == "User"`), not a bot account —
verified directly against PR #68 in `owner/repo`, where a real
Crosscheck-signed comment (posted as `hnipps`) reports `__typename: "User"`,
identically to a human reviewer's own comments on the same PR. Typename alone
cannot tell "an automated comment posted via the invoking user's token" apart
from "a human wrote this" in that case — the header/signature check is the
only reliable signal, never skip checking it. **In CI**, `pr-swarm` and
`review-triage` run under `claude-code-action` and post through its GitHub
App installation, so their own comments land as `claude[bot]`
(`__typename == "Bot"`, login ending `[bot]`) — the login-suffix/typename
check alone already classifies those correctly there; the header is still
checked for them regardless, since the same rubric runs in both places and
must not silently rely on which one it's in. `crosscheck-pr-review` is a
local, human-invoked skill only — it never runs in this repo's CI, so its
comments are always the local `User`-typename case, never the CI one. Miss
the Crosscheck signature specifically and every thread an earlier manual
`crosscheck-pr-review` run left on this PR reads as human-participated and
defers forever, never actioned or resolved — in both places this skill runs,
since Crosscheck's signature-carrying comments are never anything but
`User`-typename.

If you cannot confidently classify a participant, treat them as human. A
thread wrongly bucketed `deferred` costs one report line; a thread wrongly
auto-resolved on a human's comment costs their trust in the loop.

A thread that is human stays **untouched**: no fix pushed on its behalf, no
resolve, no reply — not even a "will look into it." Bucket it `deferred`
with `gate: human` (see `SKILL.md` Step 7's `deferred_threads` shape).

### Bot login list

Known review bots, for the login-match check above (case-insensitive
substring match on login unless marked exact):

| Login (or pattern) | Source |
| --- | --- |
| `github-actions[bot]` | GitHub Actions default token |
| `dependabot[bot]` | Dependabot |
| `greptile` (substring) | Greptile |
| `veria` (substring) | Veria |
| `coderabbit` (substring) | CodeRabbit |
| `cursor` (substring) | Cursor bot |
| `sonarcloud` (substring) | SonarCloud |
| `codescene` (substring) | CodeScene |
| `sourcery` (substring) | Sourcery |
| `ellipsis` (substring) | Ellipsis |
| `stamphog`, `posthog-code` (substring) | PostHog internal bots |
| any login containing `claude` and ending `[bot]` | Claude review GitHub Apps |

This list is not exhaustive. A login not on it is not automatically human —
apply the `__typename == "Bot"` and `[bot]`-suffix checks first, and treat a
genuinely unrecognised account as human only when none of the automated
signals fire. Extend this table when a new review bot shows up rather than
special-casing it in `SKILL.md`.

### Body-signature list (no login match — posted via a human account)

Some automated reviewers, including our own two skills, post through the
invoking user's personal GitHub account rather than a bot account, so no
login on the table above will ever match them. These are recognised by an
exact body signature instead:

| Signature (exact, checked anywhere in the comment body) | Source |
| --- | --- |
| `🤖 Automated comment by **pr-swarm**` | pr-swarm / review-triage (this plugin) |
| `_🛡️ Authored by Crosscheck 🛡️_` | `crosscheck-pr-review` |

Check this list **in addition to**, not instead of, the login table above —
a login-table bot can also carry a body signature, and a body-signature
source is never on the login table. Extend this table, the same way as the
login table, when a new human-account-posting reviewer shows up.

## 2. Reading severity off a thread

Every `pr-swarm`-originated inline comment carries its severity in the
comment body itself, in the tag line `SKILL.md` Step 6a defines:

```
**[<reviewer_tag>]** <severity emoji> <SEVERITY>
```

Parse `<SEVERITY>` directly from that line (`CRITICAL`, `HIGH`, `MEDIUM`,
`LOW`, or `NIT`; convergent findings use the same line with
`**[convergent: <a> + <b>]**` in place of the tag). Third-party bot threads
rarely carry a machine-parseable severity — fall back to the keyword
scan in *3. Classifying a non-pr-swarm bot thread* below.

## 3. Classifying a thread: actionable / nit / ambiguous

A thread is **actionable** only when **all** of the following hold:

- Severity is `CRITICAL` or `HIGH` (or the finding is convergent — flagged
  by two or more reviewers/lenses independently — which counts as
  actionable regardless of the individual severities, since convergence is
  itself the higher-confidence signal).
- The suggested fix is concrete enough that a reader knows exactly what to
  change: a specific rename, a missing null/None check, a forgotten
  `await`, an off-by-one, a missing permission check with a named check to
  add. "Consider refactoring this" is not concrete; "add an ownership check
  before this fetch" is.
- The change is localised to one file, or a small set of tightly related
  edits (e.g. a fix and its one covering test).
- Applying it does not require a new dependency, a new design decision, or
  widening the PR's stated scope.

A thread that fails only the severity bar (a real, well-scoped, concrete
`MEDIUM`/`LOW` finding) is a **nit** — resolve it, one-line reason in the
report, no code change required (a nit that happens to have an obvious
one-line fix may still be actioned at the responder's discretion, but is not
required to be).

Everything else — severity is high enough but the fix is not concrete,
touches multiple files without a tight relationship, needs a design
decision, or you are simply unsure — is **ambiguous**. Route it through the
autonomy ladder (section 4) before it can be deferred.

### Classifying a non-pr-swarm bot thread

Third-party bots (Greptile, CodeRabbit, Dependabot, etc.) rarely emit a
parseable severity tag. Apply the same three-part actionable test above
using the bot's own wording as the substitute for "severity": treat a bot's
own escalation language (`security`, `bug`, `critical`, `must fix`, "this
will break", "vulnerability") as equivalent to `CRITICAL`/`HIGH`; treat
"nit", "style", "consider", "optional", "minor" as `LOW`/nit-equivalent. A
Dependabot version-bump PR comment about a CVE is `CRITICAL`-equivalent
regardless of wording. When a bot's severity is genuinely unclear from its
own text, treat it as ambiguous rather than guessing a tier.

## 4. The autonomy ladder — before deferring an ambiguous bot/pr-swarm thread

Applies **only** to threads that passed the human-participation gate (bot or
pr-swarm-own) and landed in `ambiguous`. It never applies to a human-gated
thread — those are always deferred untouched, however trivial the change
looks; protecting the human review conversation is a different concern from
judging a fix's difficulty.

- **Just do it** — the outcome is unambiguously better and there is
  essentially one sensible way to get there (a trivial, reversible
  improvement — a typo, an unused import, a clearer name with no call-site
  ambiguity). Apply the fix, commit, push, resolve the thread, reply with
  the commit SHA and a one-line description. Bucket `promoted`.
- **Do it, but recommend and ask** — more than one reasonable solution
  exists and you picked one. Apply it, commit, push, resolve, reply — and
  the report entry (`SKILL.md` Step 7, the `recommend_and_ask` list) must
  name the alternative you didn't take, so the human can redirect cheaply
  if they'd have gone the other way. Bucket `promoted`.
- **Stop and ask** — applying the change would violate one of the four
  rules of simple design (see `xp` lens in `lenses.md` for the rule
  ordering), or it is genuinely unclear which outcome is better, or the fix
  would need a new dependency or a scope decision only the PR author can
  make. Bucket `deferred`, thread left unresolved, reason recorded.

`Stop and ask` is the only ladder outcome that defers. `Just do it` and
`recommend and ask` both resolve the thread — this mirrors the spec's
autonomy-ladder decision directly: "just-do-it and recommend-and-ask both
get applied, only stop-and-ask defers."

When in doubt between `recommend and ask` and `stop and ask`, prefer `stop
and ask` — a nagging deferred line costs a report entry; a wrong push costs
a revert and trust.

## 5. Reply vs. no-reply — who gets a reply

- **Bot and pr-swarm-own threads** (actioned, nit-resolved, or
  ladder-promoted): reply is permitted and expected — the reply carries the
  commit SHA / reason, per `SKILL.md` Step 4. These are the only threads
  `review-triage` ever replies to.
- **Human-participated threads**: never a reply from this skill. A human
  wants a response from the PR author, not from a bot pretending to be one.
  Silence here is correct behaviour, not an oversight.

## 6. Waiting on third-party bots (feeds `SKILL.md` Step 5)

The bot login list above is also the list `SKILL.md` Step 5 uses to decide
which bots to wait for after a push, within the bounded ~5 minute window.
Only bots that have historically reviewed this repo's PRs are worth waiting
for — an entry on the list with no recent activity in the target repo can
be skipped from the wait (still triaged normally if it does eventually
comment) without extending the window.
