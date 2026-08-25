# intent/

This directory holds the stage-1 artefact of Crosscheck's development framework: the intent file. It is the first thing committed for any change, before a design or an implementation exists.

## What an intent file is

An intent file states the problem, the desired outcome, who and what it touches, the constraints it must respect, and any open questions still unresolved. It is deliberately short and non-technical — it does not propose a design or an implementation, only the change worth making and why.

## Naming convention

`intent/<yyyy-mm-dd>-<slug>.md`, where the date is the day the intent was written and the slug is a short, hyphenated description of the change (e.g. `intent/2026-08-25-playbook-integration.md`).

## The rule

A change of any size starts as an intent file. Once that intent is accepted, the change gains a `spec.md` (generated from the intent, flagging concerns rather than silently resolving them) and then a `plan.md` (files that change, order of work, risks, proof/tests) before any implementation begins. Together, intent → spec → plan → diff/tests → review findings form the audit trail for the change.

Use `intent/TEMPLATE.md` as the starting point for a new intent file.

## Full picture

See `docs/assurance/DEVELOPMENT-FRAMEWORK.md` for how the full artefact chain fits together, which Crosscheck skills sit at each stage, and how tiers determine which artefacts a given change requires.
