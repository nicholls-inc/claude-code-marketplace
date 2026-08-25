# Gate: Protected-Surface Roadmap Refusal

## What this gate protects

A **protected surface** is a file that defines how a Crosscheck skill or agent behaves, or that encodes a correctness contract ("invariant") Crosscheck checks code against. Changing one of these is higher-stakes than an ordinary edit, because it can silently change what gets verified. Crosscheck's `/protected-surface-amend` skill exists to make such changes deliberate and traceable, by generating a governance-note block for every protected-surface edit — a structured record of what changed, why, and under whose authority.

One field in that record is the **governing roadmap item**: an existing, tracked entry under `docs/assurance/` that this change is fulfilling. Requiring a roadmap item stops protected-surface changes from being justified after the fact by whoever happens to be editing the file that day. If no such item exists, the skill refuses to proceed — it will not invent one, and it will not let the edit through on a weaker justification.

## What you are being asked to decide

The skill has searched `docs/assurance/` and found no roadmap item that covers this change. You must resolve this before the amendment — and therefore the protected-surface edit — can go forward. This refusal is unconditional: it applies the same way whether a human or an automated agent triggered the skill.

## Your options

- **Open a new roadmap item.** Add an entry under the appropriate horizon in `docs/assurance/` (see `docs/assurance/ROADMAP.md` for the structure), describing the change and why it is needed, then re-run `/protected-surface-amend`. Use this when the change is genuinely new work that has not been tracked anywhere yet.
- **Cite an existing item.** If this change is a corrective fix to something already tracked and landed — for example, a bug found in a previously-approved amendment — name that existing roadmap item explicitly and confirm it authorises this correction, then re-run the skill.
- **Abandon the change.** If neither applies, do not make the edit. A protected-surface change with no roadmap backing is not authorised, and proceeding anyway would bypass the gate entirely.

## What happens next in each case

- Opening or citing an item and re-running the skill lets drafting continue normally — the skill will pick up the roadmap reference and produce the rest of the governance-note block.
- Abandoning the change means no edit is made to the protected surface; the working change should be reverted or left unstaged.

## How long this takes

Usually a few minutes to write a short roadmap entry, or seconds to cite an existing one. Abandoning is immediate.

## Who to ask if unsure

The Crosscheck maintainers, via a GitHub issue on this repository.
