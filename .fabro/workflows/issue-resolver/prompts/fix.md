# Stage 6 — Fix verification failures (ONE-SHOT)

You are the implementer, on your single fix pass. The verify stage
flagged one or more failures. Your job is to address **every** failure
in **one** pass.

## You get exactly ONE chance.

- This is your only fix pass. After this stage, the workflow exits to
  a PR.
- Whatever you don't address — fix or document — reaches the PR
  unflagged.
- No new scope. Address only what verify flagged.

## What you have

Your preamble contains:

- The full **verify** stage response (failures listed across Check 1,
  Check 2, Check 3).
- `verify_concerns` in context — the itemised list verify wrote.
- The **implement** stage's response (`files_changed`, artifact paths,
  declared verification track).
- `plan.md` is on disk; the codebase is in whatever state implement
  left it.

## What to do

1. Re-read `plan.md` and the verify response so you know exactly what
   was promised and what was flagged.
2. List every concern from `verify_concerns`. Number them.
3. For each concern, do exactly **one** of:
   - **Fix** — edit the code, edit the artifact, add a missing
     artifact. Cite which file changed.
   - **Defer** — the concern is a real issue but cannot be addressed
     in this pass without scope creep or external input. State the
     concrete reason. Deferred concerns surface on the PR for human
     triage.

   Do **not** silently ignore a concern. Every numbered item gets a
   status.

4. After all fixes: re-run the verification artifacts on the declared
   track:
   - **`formal`** — `mcp__plugin_crosscheck_dafny__dafny_verify` on
     each `.dfy`.
   - **`lightweight` / `semi-formal`** — the named property tests /
     regression tests.

   Quote the new exit status / test output verbatim. Record the final
   status as `green`, `red`, or `mixed` (some artifacts green, some
   red).

5. Commit fix changes. Conventional Commits — `fix:` prefix. The
   commit message body lists fixed concerns and any deferred ones,
   so the PR description carries the verification status forward.

## Output

Markdown report with:

1. **Concern resolution table** — for each numbered concern: status
   (`fixed` | `deferred`), affected file, one-line evidence.
2. **Re-run verification output** — verbatim Dafny exit status / test
   output tail.
3. **Final status** (`green` | `red` | `mixed`) with a one-paragraph
   summary of what is and isn't verified.

End with this JSON block:

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "fix_attempted": true,
    "fix_concerns_total": <integer>,
    "fix_concerns_resolved": <integer>,
    "fix_concerns_deferred": <integer>,
    "verification_status_after_fix": "<green|red|mixed>"
  }
}
```

## Discipline

- One pass. Every concern gets a status — fixed or deferred, never
  silent.
- Surgical changes only. The verify stage's job was to surface
  failures; do not volunteer a redesign.
- Honest re-run. If your fix didn't actually make the artifact pass,
  record `red` or `mixed` — do not claim `green` you didn't observe.
- Deferred items are not failures of this stage; they are inputs to
  human triage on the PR.
