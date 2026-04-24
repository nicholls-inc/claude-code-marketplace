# FP tracker CSV — schema, append logic, kill-criterion computation

`/intent-check` appends one row to `.assurance/intent-check-fp-tracker.csv` per pipeline run. The tracker is the only persistent record of round-trip verdicts and is the input to the 30% kill criterion that decides whether Layer 5 stays online.

The schema matches the xylem tracker at `/Users/harry.nicholls/repos/xylem/docs/assurance/next/07-fp-tracker.csv` verbatim. Parity is deliberate — the plan's "adopt plan defaults" decision mandates using the xylem schema so calibration work done in one repo transfers to the next.

## Schema

Exact header (must match byte-for-byte):

```
date,invariant_touched,phase_verdict,human_verdict
```

| Column            | Type   | Written by | Description                                                                              |
|-------------------|--------|------------|------------------------------------------------------------------------------------------|
| `date`            | string | skill      | ISO calendar date, `YYYY-MM-DD`, in the committer's local timezone.                       |
| `invariant_touched` | string | skill    | Short label, e.g. `queue.md I2 (6-of-19 field mutation coverage)`. See naming below.       |
| `phase_verdict`   | enum   | skill      | `pass` when diff-checker `match=true` AND `confidence_pct>=80`; otherwise `fail`.          |
| `human_verdict`   | enum   | human      | One of `genuine`, `genuine-planted`, `partial`, `spurious`, or empty (awaiting review).   |

**`invariant_touched` naming convention.** The label must be stable enough that two runs against the same invariant produce grep-comparable strings. Follow the xylem style:

```
<module-doc>.md <invariant-id> (<short scope descriptor>)
```

Examples (from the seed fixtures):

```
queue.md I2 (failed-terminal sampling)
queue.md I2 (6-of-19 field mutation coverage)
queue.md I3 (planted — spec claim vs code violation)
queue.md I6 (linearizability depth)
queue.md I2 (ReplaceAll privileged-path coverage)
queue.md I5b (vesselSetsEquivalentIgnoringClock)
```

If the invariant has no numbered ID, use a dash-cased keyword from the prose. Avoid free-form English — grepability is the point.

**`human_verdict` semantics.**

| Value            | Meaning                                                                                            |
|------------------|----------------------------------------------------------------------------------------------------|
| `genuine`        | The phase flagged a real spec-intent mismatch. Counts against FP numerator as 0.                    |
| `genuine-planted`| Seeded mismatch fixture caught by the pipeline. Also 0 in FP numerator; used for recall tracking.  |
| `partial`        | Finding was partially correct (right area, wrong framing). Counts as 0.5 in some reports; for the   |
|                  | kill criterion, treat `partial` as NOT spurious (i.e. the pipeline is still doing useful work).     |
| `spurious`       | False positive — the phase fired but the spec+test were actually aligned. Counts as 1 in FP numerator. |
| empty (`""`)     | Awaiting human review. Ignored by the kill-criterion computation (not in numerator or denominator). |

Filling `human_verdict` is a human responsibility — `/intent-check` never writes anything other than an empty cell. A reviewer who disputes the automated verdict updates this column by editing the CSV directly (or through a PR that touches only this file).

## Example rows

```csv
date,invariant_touched,phase_verdict,human_verdict
2026-04-23,queue.md I2 (failed-terminal sampling),fail,genuine
2026-04-23,queue.md I2 (6-of-19 field mutation coverage),fail,genuine
2026-04-23,queue.md I3 (planted — spec claim vs code violation),fail,genuine-planted
2026-04-23,queue.md I6 (linearizability depth),fail,genuine
2026-04-23,queue.md I2 (ReplaceAll privileged-path coverage),fail,partial
2026-04-23,queue.md I5b (vesselSetsEquivalentIgnoringClock),fail,spurious
2026-04-24,runner.md R1 (graceful-shutdown idempotence),pass,
```

The last row shows an append from the skill — `phase_verdict=pass`, `human_verdict` empty (no reviewer has classified it yet).

## Append logic

Claude runs these steps in order:

1. **Check existence.** `test -f .assurance/intent-check-fp-tracker.csv`. If missing, create the directory if needed (`mkdir -p .assurance`) and write the exact header line followed by a newline.
2. **Read existing rows.** Load the CSV into memory (for the kill-criterion check in Step 0 of the skill).
3. **Validate header.** If the header does not match byte-for-byte, refuse to append and tell the user — the schema has drifted and merging rows could lose data. Point at this doc.
4. **Format the new row.** Use standard RFC 4180 CSV quoting. If `invariant_touched` contains a comma, double-quote the whole field and escape embedded quotes as `""`.
5. **Append.** Open in append mode; write the new row followed by a newline. Do not rewrite the whole file (avoids a race with concurrent runs, however rare in practice).
6. **Confirm.** Re-read the tail of the file and show the user the appended row.

Never mutate existing rows. If the user wants to correct an earlier `human_verdict`, they edit the CSV by hand.

## 30% kill-criterion computation

The kill criterion says: if the rolling false-positive rate over the last 14 days of entries exceeds 30%, Layer 5 is not earning its cost and the pipeline must not keep gating commits.

**Pseudocode:**

```python
from datetime import date, timedelta

def kill_criterion_fp_rate(rows: list[dict], today: date) -> tuple[float, int]:
    """
    Returns (fp_rate, window_size).
    Rows with empty human_verdict are ignored.
    Rows with human_verdict=='partial' count as NOT spurious.
    """
    cutoff = today - timedelta(days=14)
    window = [
        r for r in rows
        if date.fromisoformat(r["date"]) >= cutoff
        and r["human_verdict"].strip() != ""
    ]
    if not window:
        return (float("nan"), 0)  # unknown
    spurious = sum(1 for r in window if r["human_verdict"].strip() == "spurious")
    return (spurious / len(window), len(window))
```

Skill behaviour against this number:

| `window_size` | `fp_rate`       | Skill action                                                                              |
|---------------|-----------------|-------------------------------------------------------------------------------------------|
| 0             | N/A             | Proceed with a warning; note the tracker is empty and no kill-criterion signal is available. |
| 1–2           | any             | Proceed with a warning; sample too small to trust.                                         |
| >= 3          | <= 30%          | Proceed silently.                                                                         |
| >= 3          | > 30%           | **Refuse to run.** Emit the kill-criterion message from Step 0 of the skill.              |

Why 30%: the threshold is inherited from xylem `07-intent-check-phase.md` acceptance criteria and confirmed in `15-intent-check-calibration.md`. A pipeline that cries wolf more than ~1 time in 3 erodes trust faster than the spec-intent drift it was meant to catch.

Why 14 days: matches the xylem "2 weeks of live operation" acceptance criterion. Short enough that a recent prompt regression trips the kill quickly; long enough that a single bad day doesn't overreact.

Why ignore empty `human_verdict`: the reviewer hasn't ruled yet. Counting empty cells either way biases the rate. The cost is that a repo with zero human review will see `window_size=0` forever — that is the correct signal (no evidence either way).

## Recommended review cadence

- **Daily (when active):** reviewer triages the previous day's empty rows.
- **Weekly:** scan for trends — is any invariant generating only spurious flags? If so, the prose is probably ambiguous; consider amending it via `/protected-surface-amend` rather than re-labelling flags.
- **Before any prompt change:** snapshot the tracker so the FP-rate impact of the prompt change is measurable.

## Interoperability with xylem

If you run `/intent-check` in a repo that already inherits a xylem-shaped tracker, the rows are directly concatenatable — the columns, types, and enum values are identical. This is the primary benefit of parity.
