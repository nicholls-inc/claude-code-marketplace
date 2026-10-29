## PARAMETERS  (the only section that changes between repos and revisions)
JOB_NAME:          audit-lane
VARIANT:           parity                 # parity | logs
OUTCOME:           Real, reproducible gaps become well-formed issues the fix chain can take, with no noise.
METRIC:            issues fixed / issues filed; false-positive rate = not planned / filed
MODE:              act                    # dry_run = audit and record, file nothing
SURFACES:          <list, ~10, rotated one per run>        # parity
REFERENCE:         <owner/repo@branch and paths>            # parity
LOG_QUERIES:       <named aggregate queries>                # logs
WINDOW:            6h                     # logs
BASELINE:          72h                    # logs
NOISE_FLOOR:       <min samples per key; ignore success-only slowness under X%>
MAX_NEW_ISSUES:    3
MAX_COMMENTS:      4
ISSUE_LABEL:       <label the fix chain picks up>
STATE_ISSUE:       <number of a pinned "audit-lane state" issue; its body holds the watermark>
RECORD_SINK:       run_summary

## CONTRACT  (stable; changed only by a human-applied revision)
Every run ends in exactly one of ACTED(summary), NOOP(reason) or BLOCKED(reason), plus one record.

1. Watermark. Read the fenced JSON under "## lever state" in STATE_ISSUE's body:
   {"source_sha", "reference_sha", "window_end", "next_surface"}. Missing or unparseable: treat this as
   the first run and say so. If nothing moved: write state, NOOP(no change), post nothing.
2. Pick the slice. Parity: the next surface in rotation, plus any surface whose files changed since
   the watermark. Logs: the latest WINDOW vs BASELINE, per stage, outcome and role.
3. Audit read-only. Parity: compare behavior in source on both sides (states, accessibility,
   responsive layout, loading, error and empty states). Logs: run only aggregate queries; never
   copy raw log lines or secrets.
4. Keep a finding only if it is reproducible with evidence: file:line on both sides, or the query plus
   baseline vs current numbers above NOISE_FLOOR.
5. Dedup. Search open issues and PRs with ISSUE_LABEL. Skip any key closed as not planned. Skip a
   finding if an open PR already touches its files. Prefer a comment on an existing issue to a new one.
6. File at most MAX_NEW_ISSUES issues and MAX_COMMENTS comments, labeled ISSUE_LABEL. Each issue has a
   one-line key, the evidence, the expected behavior and a suggested acceptance check.
7. Never change code, open PRs, close issues or touch credentials.
8. Every run, silent ones included, ends by rewriting the "## lever state" block with
   github/update_issue_body: the new SHAs or window end, and the next surface in rotation. Leave the rest of
   the body alone. If MODE=dry_run, print the new block instead.
