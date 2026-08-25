# Intent: Adopt the playbook as Crosscheck's development framework and make every gate self-explanatory

## Problem statement
Crosscheck's own development has no artefact chain governing it — there is no committed intent, spec, or plan convention, so changes to Crosscheck itself are not held to the same rigour the plugin asks of the codebases it verifies. Separately, Crosscheck's many human gates (sign-offs, triage prompts, refusal messages, hook blocks) are opaque to newcomers: each was written for someone who already knows Crosscheck's internals, with no plain-language explanation of what is being protected, what the options mean, or who to ask.

## Proposed outcome
Crosscheck adopts Anthropic's AI-native SDLC playbook as its own development framework: every change starts as an `intent.md`, gains a `spec.md` and a `plan.md` before implementation, and is verified against a tiered artefact requirement enforced in CI. Every existing human gate gets a plain-language explainer under `docs/gates/` and emits a short, uniform "Action needed" message at the point it is delivered, so a newcomer can understand and act on any gate without prior context.

## Affected users and systems
Crosscheck contributors (this repository's own development), users of Crosscheck's skills and agents (who see the gate messages at runtime), and CI (new workflows enforcing tiered artefact requirements and protected-surface checks). No end-user-facing behaviour of the verification engines themselves changes.

## Constraints
Additive only: no gate is weakened, removed, or bypassed; no kill criterion, threshold, or validity invariant is relaxed; no skill's verification logic changes. Edits to `SKILL.md` or `crosscheck/agents/*.md` touch only output/message templates and documentation. Every new document stays under two pages. All prose uses British spelling. No model identifiers appear in any committed file. Gate messages follow the fixed three-sentence "Action needed" format with a link to `docs/gates/<file>.md`, with the repo URL derived from the git remote rather than hard-coded.

## Open questions
None — resolved in `plan.md`.
