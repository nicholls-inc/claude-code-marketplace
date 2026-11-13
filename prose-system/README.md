# AI prose system: handoff package

Start with `HANDOFF.md`.

To start a Claude Code session on it, put this folder in a repository and give the agent this instruction:

> Read HANDOFF.md and the files it lists, in the order it gives. Then tell me what you plan to do first and which open questions you need answered. Do not generate any text or build anything until I reply.

Contents:

- `HANDOFF.md`: status, tasks, rules, stop points and caveats
- `BRIEF.md`: the original brief, unchanged
- `FINDINGS.md`: the investigation write-up
- `EXPERIMENTS.md`: eight experiments, none run
- `EVALUATION.md`: reader protocol, brief sets, judge calibration
- `OPEN_QUESTIONS.md`: questions for the owner
- `scripts/profile.py`: surface profile of text samples
- `scripts/brown_reference.py`: reproduces the human reference measurement
- `reference/brown_reference.json`: its output
