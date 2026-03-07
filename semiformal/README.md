# semiformal — Semi-formal Reasoning for Claude Code

A Claude Code plugin that implements semi-formal reasoning from the "Agentic Code Reasoning" paper (Ugare & Chandra, Meta, 2026). Semi-formal reasoning is a structured prompting methodology that requires constructing explicit premises, tracing execution paths, and deriving formal conclusions. Unlike unstructured chain-of-thought, semi-formal reasoning acts as a certificate: the agent cannot skip cases or make unsupported claims.

## Key Results from the Paper

- **Patch equivalence**: 78% to 88% accuracy (curated), 93% on real-world patches
- **Code question answering**: 87% accuracy on RubberDuckBench (+9pp over standard)
- **Fault localization**: +5-12pp over standard reasoning

## Usage

### Skills

Five independent skills for granular control:

| Skill | Description |
|-------|-------------|
| `/reason` | General-purpose semi-formal reasoning for any code question |
| `/analyze-code` | Deep code question answering with function trace tables and data flow analysis |
| `/compare-patches` | Patch equivalence verification with structured proof/counterexample |
| `/locate-fault` | Fault localization using 4-phase PREMISE -> CLAIM -> PREDICTION chain |
| `/trace-execution` | Hypothesis-driven execution path tracing |

### Orchestrator Agent

For end-to-end workflows, use the `reasoning-orchestrator` agent which automatically classifies your problem and routes to the appropriate skill.

## Core Principles

1. Every claim must cite file:line evidence
2. Always read actual code — never guess from function names
3. Check alternative hypotheses before concluding
4. The structured format IS the reasoning process, not just output formatting
5. Name resolution matters — check for shadowing at every scope

## When to Use

- **Code review**: "Is this refactor safe?"
- **Bug hunting**: "Why does this test fail?"
- **Understanding code**: "What does this function actually do?"
- **Comparing implementations**: "Are these two approaches equivalent?"
- **Verifying claims**: "Is this code thread-safe?"
