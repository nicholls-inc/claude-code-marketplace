# Agent Harness: User Stories

**Perspective:** A software engineer using the harness to complete software engineering work.

**Companion document:** *The Definitive Guide to Harness Engineering for AI Agent Systems*

---

## Terminology

| Term | Definition |
|------|-----------|
| **Mission** | An overarching user-defined work item — a GitHub issue, a Linear ticket, a CLI submission. The top-level unit of work the engineer defines. |
| **Task** | A decomposed unit of work within a mission. Created by the harness when it breaks a mission into smaller pieces. |
| **Harness** | The entire execution environment around the LLM — tools, context, memory, feedback loops, constraints, and control systems. |
| **Deterministic intermediary** | An architectural component that validates all agent actions crossing the sandbox boundary. Agents specify structured intents; the intermediary checks permissions and policies before executing. |
| **Behavioral signals** | Lightweight heuristics computed directly from agent traces without LLM involvement — repetition detection, efficiency scores, tool failure rates, etc. Used as gates to determine if expensive LLM evaluation is warranted. |
| **Sprint contract** | An agreement between generator and evaluator (or engineer and harness) on what "done" looks like before work begins. Stored locally and posted to the originating platform. |
| **Inner zone** | The agent's sandboxed workspace where it operates with full autonomy. |
| **Outer zone** | External services, persistent state, and harness runtime — all actions here require validation through the deterministic intermediary. |

---

## Cross-Cutting Principles

These principles apply across all sections. Individual stories may reference them, but they are stated here once as foundational constraints on the entire harness design.

### P.1 — Zero-Config Sensible Defaults

As an engineer, I want every harness configuration — cost limits, model selection, evaluation intensity, security levels, compaction thresholds, signal thresholds, retry limits, orchestration patterns, entropy management cadence — to ship with sensible defaults that work out of the box, derived from repo analysis, mission complexity, model capabilities, and accumulated usage data. I should be able to run my first mission without configuring anything beyond the initial bootstrap. All defaults must be inspectable, explainable (why this default was chosen), and overridable at the org, repo, mission type, or individual mission level. The harness gets smarter over time by adjusting defaults based on historical performance, but never changes a default I've explicitly overridden.

### P.2 — Originating Platform Communication

As an engineer, I want all harness-to-human interactions — status updates, clarifying questions, approval requests, sprint contracts, knowledge gap alerts, escalations — to be surfaced on the platform where the mission originated (GitHub issue comment, Linear comment, CLI prompt, etc.), so that I never have to context-switch to a separate harness UI to stay informed or unblock work. The harness meets me where I already am.

### P.3 — Deterministic Intermediary for Outer-Zone Actions

As an engineer, I want all agent actions that cross the sandbox boundary — writing to external services, updating persistent state, modifying harness runtime, changing mission status, posting to platforms — to go through a deterministic intermediary that validates permissions and enforces policies before execution. Agents specify *what* they want to do as structured intents; the intermediary decides *whether* they're allowed and executes on their behalf. This is an architectural guarantee, not a prompt-based instruction.

### P.4 — Signal-Gated Evaluation

As an engineer, I want the harness to use lightweight behavioral signals (computed without LLM involvement) as the first-pass assessment of agent health, quality, and progress. Expensive LLM-based evaluation is invoked only when signals cross configurable warning thresholds. Critical signal thresholds halt execution and alert me immediately without waiting for LLM analysis. Signals tell the harness *where to look*; LLM evaluation tells it *what went wrong*.

### P.5 — Automation-First with Human Approval

As an engineer, I want the harness to do the work — proposing, generating, fixing, refining — and present results for my approval, rejection, or refinement. My role is reviewing and steering, not filling in templates or manually executing steps. The harness proposes; the engineer disposes.

### P.6 — Two-Zone Security Model

As an engineer, I want security enforced through two architectural zones: an **inner zone** (the agent's sandboxed workspace) where agents operate with full autonomy and minimal friction, and an **outer zone** (external services, persistent state, harness runtime) where all actions require validation through the deterministic intermediary. Security is structural, never dependent on prompt instructions or agent compliance.

---

## Section 1: Project Bootstrapping & Configuration

**1.1** — As an engineer, I want to run a single bootstrap command that analyzes my repo and automatically generates the full harness structure (AGENTS.md, docs/ directory, progress files, feature list, entry point scripts), inferring defaults from the existing codebase (language, framework, test runner, build tools), so that my project is agent-ready with minimal manual input. *(Applies P.1, P.5)*

**1.2** — As an engineer, I want the bootstrap to auto-generate an AGENTS.md (~100 lines) that acts as a table of contents, populated with pointers inferred from my actual repo structure, so that I'm editing a smart draft rather than writing from scratch. *(Applies P.5)*

**1.3** — As an engineer, I want to define a multi-level instruction hierarchy (org-level → repo-level → directory-level), where the bootstrap detects existing conventions (eslint configs, editorconfig, CI pipelines) and auto-populates directory-level instructions from them, so that agents inherit real project standards without me transcribing them manually. *(Applies P.1, P.5)*

**1.4** — As an engineer, I want to run a legibility audit that scores my repo across seven dimensions (bootstrap self-sufficiency, task entry points, validation harness, linting/formatting, codebase map, doc structure, decision records), and for each gap, the harness offers to fix it automatically — generating missing scripts, scaffolding missing docs, wiring up linters — prompting me with targeted questions only when it can't infer the right answer, so that remediation is a guided conversation, not a to-do list I handle alone. *(Applies P.5)*

**1.5** — As an engineer, I want the bootstrap to generate a structured docs/ directory pre-populated with drafts inferred from my codebase (e.g., a design doc skeleton pulled from code comments, a domain guide seeded from README content), so that agents have usable progressive-disclosure docs from the start rather than empty templates. *(Applies P.5)*

**1.6** — As an engineer, I want the bootstrap to detect my current tech stack and auto-configure preferred technologies and patterns, flagging any dependencies that are known to be hard for agents to model and suggesting "boring" alternatives, so that I make informed choices without researching agent compatibility myself. *(Applies P.1, P.5)*

**1.7** — As an engineer, I want the bootstrap to extract real code examples from my repo that demonstrate preferred style (naming, error handling, test structure) and embed them in the harness configuration automatically, so that agents learn from my actual codebase rather than generic descriptions. *(Applies P.5)*

**1.8** — As an engineer, I want the bootstrap to discover, validate, and if necessary fix the build, test, lint, and run commands — attempting auto-repair of broken scripts and asking me for input only when it can't resolve issues on its own — so that agents have verified, working task entry points from day one. *(Applies P.5)*

**1.9** — As an engineer, I want the entire bootstrap to run as an interactive session where the harness does the work and I approve, reject, or refine its proposals, rather than filling in blanks in a template, so that setup feels like a code review, not a questionnaire. *(Applies P.5)*

---

## Section 2: Mission Definition & Delegation

**2.1** — As an engineer, I want to describe a mission in natural language and have the harness automatically determine whether it's simple enough for a single agent or complex enough to require multi-agent orchestration, so that I don't have to make architectural decisions about delegation for every mission. *(Applies P.1)*

**2.2** — As an engineer, I want the harness to generate a sprint contract (what "done" looks like, what will be built, how success is verified) and present it on the platform where the mission originated — as a GitHub issue comment, a Linear comment, a CLI response, etc. — AND store the contract locally as a structured artifact the harness can reference throughout execution, so that alignment happens in my existing workflow and the harness always has the contract at hand. *(Applies P.2, P.5)*

**2.3** — As an engineer, I want to constrain missions by specifying deliverables and required verification steps — including formal verification outputs (e.g., a Dafny proof file) or semi-formal reasoning certificates — while leaving the agent freedom in how it arrives at those outputs, so that I control the rigor of the result without micromanaging the implementation path.

**2.4** — As an engineer, I want all agent interactions with external systems (creating PR comments, updating issue status, writing to feature lists) to go through the deterministic intermediary: the agent specifies *what* it wants to do as a structured intent, the intermediary verifies the agent has permission and the action complies with policy, and only then executes the action on the agent's behalf, so that tamper-resistance and access control are enforced architecturally, not by trusting the agent to follow instructions. *(Applies P.3)*

**2.5** — As an engineer, I want the harness to assign the right planner altitude automatically — keeping planners high-level and preventing them from specifying granular technical details — so that implementation errors in the plan don't cascade into downstream agent work. *(Applies P.1)*

**2.6** — As an engineer, I want to provide a persona and scope for the mission (e.g., "you are a test engineer who writes tests for React components and never modifies source code") and have the harness enforce those boundaries throughout execution, so that agents stay in their lane without me monitoring continuously.

**2.7** — As an engineer, I want to attach relevant context to a mission — linking to specific docs, design decisions, or code examples — and have the harness use progressive disclosure to feed them to the agent at the right moments rather than dumping everything up front, so that the agent gets high-signal context without window bloat.

**2.8** — As an engineer, I want to set explicit limits on a mission (max retries, max token budget, time budget, blast radius of files the agent may touch), with the harness providing sensible defaults I can override, so that runaway agents are halted before they waste resources or cause damage. *(Applies P.1)*

**2.9** — As an engineer, I want to define mission queues with explicit dependencies and priorities, sourced from external tools like GitHub Issues or Linear, using a standard schema (or a well-defined harness-native schema if no adequate standard exists) so that the harness can pull work items, respect dependency ordering, carry forward state between missions, and I can manage backlogs in the tools I already use.

**2.10** — As an engineer, I want the harness to automatically attempt to refine underspecified missions by consulting existing repo context (docs, code, decision records, related issues) — staying grounded, making minimal assumptions, and never making design decisions unless explicitly stated — and only escalate to me on the originating platform (GitHub comment, Linear comment, CLI prompt) if the mission remains genuinely ambiguous after automated refinement, so that most ambiguity is resolved without blocking on me. *(Applies P.2, P.5)*

---

## Section 3: Context Management

**3.1** — As an engineer, I want the harness to automatically determine the smallest, highest-signal set of context for each agent step — pulling from docs, code, memory, and tools via progressive disclosure — so that I don't have to manually curate what the agent sees and the context window stays focused. *(Applies P.1)*

**3.2** — As an engineer, I want the harness to maintain a structured entry point (AGENTS.md as table of contents) and teach agents to navigate deeper into the docs/ directory on demand, rather than pre-loading everything, so that agents behave like engineers who know where to look rather than engineers who've memorized the entire wiki.

**3.3** — As an engineer, I want the harness to use just-in-time context retrieval — maintaining lightweight identifiers (file paths, queries, links) and dynamically loading data via tools only when the agent actually needs it — so that the context window isn't consumed by content that might never be relevant.

**3.4** — As an engineer, I want the harness to run automatic compaction when context utilization crosses a configurable threshold (default: 95%), preserving architectural decisions, unresolved bugs, and implementation details while discarding redundant tool outputs and verbose intermediate reasoning, so that agents can continue working without hitting window limits or losing critical information. *(Applies P.1)*

**3.5** — As an engineer, I want to configure whether the harness uses compaction (summarize in place) or full context resets (clear window with structured handoff) based on the model in use and the mission duration, with the harness selecting an appropriate default per model, so that I can mitigate context anxiety on models prone to it without paying the overhead of resets on models that don't need them. *(Applies P.1)*

**3.6** — As an engineer, I want the harness to detect and prevent the four context failure modes — poisoning (hallucination propagating through reasoning), distraction (too much context overwhelming the signal), confusion (superfluous context influencing responses unexpectedly), and clash (contradictory context) — by validating context coherence before injection, so that the agent reasons from clean inputs.

**3.7** — As an engineer, I want the harness to apply RAG over tool descriptions when the agent has access to many tools, selecting only the relevant subset for each step rather than injecting all tool definitions, so that tool metadata doesn't crowd out task context.

**3.8** — As an engineer, I want the harness to build context through named, ordered processors — not ad-hoc string concatenation — so that context assembly is observable, testable, and I can inspect exactly what went into the window at any step.

**3.9** — As an engineer, I want the harness to separate durable state (session data, progress files, feature lists) from per-call working context (current step's relevant code, docs, tool outputs), so that persistent knowledge doesn't get accidentally compacted or lost during window management.

**3.10** — As an engineer, I want the harness to track context utilization metrics over time — window fill rate, compaction frequency, retrieval hit rate, tokens spent on tool outputs vs. task reasoning — so that I can identify context waste and tune the configuration based on real data rather than guesswork.

**3.11** — As an engineer, I want the harness to automatically select the right context strategy (Write, Select, Compress, Isolate) for each situation — writing to scratchpads when information must persist, selecting via retrieval when information exists externally, compressing when the window is filling, isolating into sub-agents when tasks are context-heavy — so that I get optimal context management without hand-picking strategies per step. *(Applies P.1)*

---

## Section 4: Tool & Integration Management

**4.1** — As an engineer, I want the harness to maintain a centralized tool catalog where each tool has a clear description, explicit parameter types, return formats, error conditions, and usage guidance — automatically validated for clarity and completeness — so that tool selection is unambiguous for both agents and humans.

**4.2** — As an engineer, I want to register tools via MCP (Model Context Protocol) as the default integration standard, so that tools are portable across models and frameworks without custom wiring per provider.

**4.3** — As an engineer, I want the harness to detect and flag overlapping or ambiguous tools in the catalog — tools whose descriptions or functionality are too similar — and suggest consolidation or disambiguation, so that agents aren't confused by redundant options. *(Applies P.5)*

**4.4** — As an engineer, I want the harness to enforce token-efficient tool outputs by default — stripping verbose metadata, returning only relevant fields, and summarizing large payloads before injection into context — so that tool results don't crowd out task reasoning. *(Applies P.1)*

**4.5** — As an engineer, I want the harness to restrict computer-use models exclusively to UI validation and testing (screenshot comparison, DOM inspection, end-to-end user flow verification), and never use them as a general tool integration strategy for legacy systems, so that agents aren't slowed down by flaky screen interactions. For legacy systems without APIs, the harness should flag them as requiring manual integration and suggest options (writing a thin API wrapper, using CLI adapters, or scripting direct database/file access). *(Applies P.5)*

**4.6** — As an engineer, I want the harness to support just-in-time tool loading — only injecting tool definitions into context when they're relevant to the current step (via RAG over descriptions or explicit scoping) — so that the full catalog doesn't consume context when only a few tools are needed.

**4.7** — As an engineer, I want custom lint rules and CI checks to function as implicit tools that teach the agent while it works — injecting remediation instructions directly into agent context when violations are detected — so that the agent learns from failures in real time without me writing additional guidance.

**4.8** — As an engineer, I want the deterministic intermediary to validate only actions that affect external services (PRs, issue comments, deployments, database writes) or persistent files outside the agent's sandboxed workspace. Within their own ephemeral workspace, agents must have full flexibility to use tools freely — reading, writing, running code, invoking local commands — so that security controls don't create friction on the inner loop of actual work. *(Applies P.3, P.6)*

**4.9** — As an engineer, I want agents to operate within sandboxed workspaces with network restrictions and Claude Code settings controlling permitted actions, so that CodeAct and other execution patterns can run freely without per-invocation validation. The sandbox is the security boundary — anything inside it is the agent's domain, anything that crosses it goes through the intermediary. *(Applies P.6)*

**4.10** — As an engineer, I want to define per-tool permission scopes (read-only, write-with-approval, full-autonomy) and assign different permission sets to different agent roles, so that a test-writing agent can read source code but not modify it, while a refactoring agent can modify code but not deploy it.

**4.11** — As an engineer, I want the harness to track tool usage metrics — call frequency, failure rates, average token cost per tool, latency — so that I can identify underperforming tools, remove unused ones, and optimize the catalog based on real usage data.

**4.12** — As an engineer, I want the harness to gracefully handle tool failures — retrying with backoff for transient errors, falling back to alternative tools where configured, and surfacing clear error context to the agent rather than raw stack traces — so that a single tool failure doesn't derail an entire mission. *(Applies P.1)*

**4.13** — As an engineer, I want to be able to add a new tool to the catalog by pointing the harness at an API spec (OpenAPI, GraphQL schema, MCP manifest) and having it auto-generate the tool registration with description, parameters, and error handling, asking me to confirm or refine, so that tool onboarding is minutes, not hours. *(Applies P.5)*

**4.14** — As an engineer, I want the harness to enforce a clear two-zone security model: an inner zone (the agent's sandboxed workspace) where agents operate with full autonomy and minimal friction, and an outer zone (external services, persistent state, harness runtime) where all actions require validation through the deterministic intermediary — so that security is tight where it matters without strangling the agent's ability to do its job. *(Defines P.6)*

---

## Section 5: Memory & State Persistence

**5.1** — As an engineer, I want the harness to automatically maintain a progress file per mission, scoped to the mission's identity (GitHub issue number, Linear ticket ID, CLI submission ID), updated after each meaningful unit of work — what was completed, what failed, what's next — so that parallel missions stay isolated, agents stay focused on their mission's state, and I can review progress per mission independently.

**5.2** — As an engineer, I want the harness to use git commits as primary checkpoints, with descriptive commit messages generated after each completed feature or meaningful change, so that every state is recoverable, the audit trail is built into version control, and the next session can reconstruct context from git history. *(Commit messages go through the deterministic intermediary as outer-zone actions — applies P.3)*

**5.3** — As an engineer, I want the harness to maintain a structured feature list (in JSON) per mission that tracks pass/fail status for every deliverable, where status changes go through the deterministic intermediary to prevent agents from gaming progress, so that I always have a trustworthy view of actual completion state. *(Applies P.3)*

**5.4** — As an engineer, I want the harness to support three distinct memory types — procedural (rules, conventions, system prompts), semantic (learned facts, knowledge graph entries, cross-session discoveries), and episodic (examples of desired behavior, past interaction patterns) — each stored and retrieved differently, so that the right kind of memory is available in the right context.

**5.5a** — As an engineer, I want all persistent knowledge required by agents to be repository-local and versioned — never siloed in Slack threads, Google Docs, or people's heads — so that from the agent's perspective, everything it needs is accessible in-context.

**5.5b** — As an engineer, I want the harness to detect when a mission references knowledge that isn't captured in the repo — undefined domain terms, external specs, unlinked design decisions, references to conversations — and block the mission from starting until the gap is resolved, so that agents never begin work on an incomplete foundation.

**5.5c** — As an engineer, I want the harness to resolve knowledge gaps through a structured flow: first, attempt automated resolution by searching the repo, linked docs, and connected integrations (Google Drive, Confluence, GitHub wikis) for the missing information; if found, propose codifying it into the repo for my approval. If not found, surface the specific gap on the originating platform with a clear description of what's missing and what format it's needed in, so that I can provide the missing context in-place and the harness can ingest it and unblock the mission. *(Applies P.2, P.5)*

**5.5d** — As an engineer, I want to provide missing context in whatever form is natural — pasting text into an issue comment, linking a document, uploading a file, or answering targeted questions from the harness — and have the harness normalize it into the repo's knowledge structure (appropriate docs/ location, decision record, domain guide entry), so that ad-hoc knowledge contributions become durable, version-controlled artifacts automatically. *(Applies P.5)*

**5.6** — As an engineer, I want the harness to generate structured handoff artifacts when a session ends or a context reset occurs — summarizing what was completed, what failed, unresolved decisions, and recommended next steps — so that the next session picks up cleanly without re-discovering state.

**5.7** — As an engineer, I want the harness to maintain a session-level key-value store for structured data (configuration state, intermediate results, computed values) that is distinct from conversation history, so that agents can read and write structured state without it being subject to compaction or lost in conversation noise.

**5.8** — As an engineer, I want the harness to enforce memory isolation between missions, between agents, and between users, so that cross-contamination between missions is prevented and a compromised agent session cannot access state belonging to other missions or users. *(Applies P.6)*

**5.9** — As an engineer, I want the harness to validate and sanitize all data before it enters memory systems — checking for hallucinated content, contradictions with existing state, and malicious payloads — so that context poisoning doesn't propagate across sessions.

**5.10** — As an engineer, I want the harness to implement a getting-up-to-speed ritual at the start of every session — automatically reading the mission's progress file, recent git history, feature list status, and any handoff artifacts — so that agents orient themselves consistently without me scripting the onboarding each time. *(Applies P.1)*

**5.11** — As an engineer, I want the harness to store execution plans, completed plans, and known technical debt as first-class versioned artifacts in the repository, so that agents can reference past decisions and outstanding debt when planning new work.

**5.12** — As an engineer, I want the harness to support scratchpads — ephemeral, agent-writable notes that persist outside the context window within a session but are discarded at session end unless explicitly promoted to durable state — so that agents can think in long-form without consuming context, while I control what survives across sessions.

---

## Section 6: Evaluation & Quality Assurance

**6.1** — As an engineer, I want the harness to enforce generator-evaluator separation by default — the agent that builds the work never evaluates its own output — so that evaluation is structurally independent rather than relying on an agent to be self-critical (which models are demonstrably bad at). *(Applies P.1)*

**6.2** — As an engineer, I want the evaluator to test the live running application — using browser automation (Playwright) to click through UI features, hit API endpoints, and verify database states — not just read the code, so that evaluation reflects what a real user would experience.

**6.3** — As an engineer, I want the harness to support formal verification and semi-formal reasoning as evaluation steps — requiring Dafny proof files, reasoning certificates, or other verification artifacts as mission deliverables — so that correctness is proven, not just tested.

**6.4** — As an engineer, I want to calibrate the evaluator by reviewing its judgment logs, identifying where it diverges from my judgment, and updating its prompt with few-shot examples that include detailed score breakdowns, so that the evaluator aligns with my quality standards over time rather than applying generic criteria.

**6.5** — As an engineer, I want each mission's sprint contract to define hard evaluation thresholds per criterion, where failing any threshold fails the sprint and returns detailed feedback to the generator with specific issues to fix, so that the bar for "done" is explicit and enforced, not negotiable.

**6.6** — As an engineer, I want the evaluator's pass/fail decisions and status updates to go through the deterministic intermediary, so that an evaluator cannot mark work as complete without the action being validated and the feature list updated through the controlled path. *(Applies P.3)*

**6.7** — As an engineer, I want the harness to support iterative generator-evaluator loops — the generator builds, the evaluator critiques with specific feedback, the generator revises, the evaluator re-evaluates — with a configurable maximum number of iterations (sensible default provided) before escalating to me, so that quality improves through structured iteration without infinite loops. *(Applies P.1, P.2)*

**6.8** — As an engineer, I want the harness to treat evaluation cost as a variable — applying full generator-evaluator loops for missions beyond what the current model handles reliably solo, and using signal-gated lightweight checks (linting, type checking, test pass, behavioral signals) for well-understood missions — so that I'm not paying evaluator overhead on trivial work. *(Applies P.1, P.4)*

**6.9** — As an engineer, I want the harness to maintain a ground truth dataset (50+ expected interactions/outputs per mission type) that evaluations are benchmarked against, and alert me when agent output quality drifts from established baselines, so that I catch regressions before they compound.

**6.10** — As an engineer, I want the harness to run all existing tests, linters, type checks, and CI pipeline checks as a mandatory evaluation step before any mission is marked complete, so that mechanical verification is never skipped regardless of how confident the agent or evaluator is.

**6.11** — As an engineer, I want the harness to produce a quality scoring document per mission that grades each deliverable and tracks quality trends over time, updated automatically, so that I have a longitudinal view of agent output quality without manually auditing every mission.

**6.12** — As an engineer, I want the harness to pass the future-proofing test: if I swap in a more powerful model without changing the harness, mission quality should improve, not degrade, so that I know the harness is measuring real quality rather than rewarding model-specific quirks.

**6.13** — As an engineer, I want the harness to detect when the evaluator is "talking itself into approving" mediocre work — using behavioral signals (e.g., evaluator flagging issues then immediately reversing judgment) as a heuristic gate, escalating to me only when the pattern is detected — so that the known failure mode of LLM-as-judge is caught structurally without requiring LLM meta-evaluation of every judgment. *(Applies P.4)*

---

## Section 7: Multi-Agent Orchestration

**7.1** — As an engineer, I want the harness to automatically select the right orchestration pattern for each mission — sequential pipeline, parallel ensemble, orchestrator-workers, or handoff — based on the mission's complexity, domain spread, and tool requirements, so that I get the right architecture without manually designing agent topologies. *(Applies P.1)*

**7.2** — As an engineer, I want sub-agents to act as context firewalls — the dispatching agent sees only the prompt it sent and the condensed summary returned (1,000–2,000 tokens), never the sub-agent's intermediate tool calls or reasoning — so that the parent agent stays in the "smart zone" and its context window isn't polluted by delegated work.

**7.3** — As an engineer, I want agents to communicate through files (specs, progress docs, requirements, contracts) rather than message passing, so that inter-agent communication is inspectable, versioned, and faithful to specifications without me tracing opaque internal message flows.

**7.4** — As an engineer, I want the harness to split a mission into multi-agent when specific conditions are met — complex logic with many conditionals, tool overload from similar/overlapping tools, or tasks spanning distinct domains — and default to single-agent for well-defined missions, so that multi-agent overhead (up to 15× more tokens) is only incurred when it produces measurably better results. *(Applies P.1)*

**7.5** — As an engineer, I want the harness to support an agent-to-agent review loop — where one agent reviews another's changes, provides feedback, the author agent revises, and the cycle continues until the reviewer is satisfied or a max iteration limit is hit — so that quality review happens continuously without requiring my attention on every change. *(Applies P.1)*

**7.6** — As an engineer, I want the harness to avoid over-constraining which sub-agents can access which tools — giving sub-agents broad tool access within their sandbox by default — so that "tool thrash" from micro-optimized access controls doesn't degrade results. *(Applies P.6)*

**7.7** — As an engineer, I want each sub-agent to have its own isolated context window scoped to its specific task, inheriting only the relevant slice of mission context (the sprint contract, its assigned deliverables, applicable docs), so that sub-agents get focused context rather than the parent's entire accumulated state.

**7.8** — As an engineer, I want the harness to support parallel sub-agents that work the same task simultaneously and produce independent outputs that are compared or voted on, so that I can use ensemble reasoning for high-stakes deliverables where a single agent's output isn't trustworthy enough.

**7.9** — As an engineer, I want all cross-agent actions — one agent triggering another, passing results, updating shared state — to go through the deterministic intermediary, so that multi-agent coordination is auditable, policy-compliant, and no agent can unilaterally affect another agent's mission state. *(Applies P.3)*

**7.10** — As an engineer, I want the harness to track per-agent token usage, wall-clock time, and task completion rate within a multi-agent mission, so that I can identify bottleneck agents, wasteful delegation patterns, and optimize orchestration based on actual performance data.

**7.11** — As an engineer, I want the harness to gracefully handle sub-agent failure — if a sub-agent fails, times out, or exhausts its budget, the orchestrator should capture what was completed, surface the failure context, and either retry with fresh context, reassign to another agent, or escalate to me on the originating platform — so that one agent's failure doesn't silently stall the entire mission. *(Applies P.1, P.2)*

**7.12** — As an engineer, I want the harness to provide a clear visualization of the active agent topology for any running mission — which agents are active, what each is working on, their context utilization, signal health, and the flow of information between them — so that I can understand and debug multi-agent execution without reading raw logs.

---

## Section 8: Security & Trust Boundaries

**8.1** — As an engineer, I want the harness to enforce the two-zone security model as a foundational architectural constraint — inner zone (sandboxed workspace, full agent autonomy) and outer zone (external services, persistent state, harness runtime, all actions validated through the deterministic intermediary) — so that security is structural, not dependent on prompt instructions or agent compliance. *(Defines P.6)*

**8.2** — As an engineer, I want agent sandboxes to run in fully isolated environments (Docker containers, VMs, or equivalent), with network access restricted to an allowlist of approved domains, so that a compromised or misbehaving agent cannot reach arbitrary external services or exfiltrate data.

**8.3** — As an engineer, I want to define per-tool permission scopes (read-only, write-with-approval, full-autonomy) and assign permission sets per agent role, enforced by the deterministic intermediary, so that a test agent can read source but not modify it, a coding agent can modify code but not deploy it, and a deploy agent can deploy but not modify code. *(Applies P.3)*

**8.4** — As an engineer, I want the harness to require human-in-the-loop approval for a configurable set of high-impact actions — production deployments, database writes, financial transactions, irreversible deletions, and any action I designate — surfacing approval requests on the originating platform with full context of what the agent wants to do and why, so that I retain a hard veto on consequential actions. *(Applies P.2, P.3)*

**8.5** — As an engineer, I want the harness to treat prompt injection as an unsolved problem and never rely on prompt-based defenses for security — all security boundaries must be architectural (sandboxing, the intermediary, permission scoping, network restrictions) — so that security holds even if the model is successfully manipulated.

**8.6** — As an engineer, I want the harness to enforce structured output schemas (strict JSON) for all agent-to-intermediary communication, so that agents cannot embed unexpected content, escape their intent format, or bypass policy validation through freeform text. *(Applies P.3)*

**8.7** — As an engineer, I want all agent actions that cross the sandbox boundary to be logged with full context — what was requested, what policy was evaluated, whether it was approved or denied, who approved it (human or automated rule), and the result — so that I have a complete, tamper-proof audit trail for every external interaction.

**8.8** — As an engineer, I want the harness to enforce that instructions come from the harness configuration (system prompts, AGENTS.md, sprint contracts), never from untrusted user input interpolated directly into agent instructions, so that external input cannot hijack agent behavior.

**8.9** — As an engineer, I want to configure safety levels per mission or per agent role — ranging from conservative (all external actions require approval) to autonomous (only designated high-impact actions require approval) — with sensible defaults based on mission complexity, so that I can dial trust up or down based on the stakes of the work and my confidence in the harness configuration. *(Applies P.1)*

**8.10** — As an engineer, I want the harness to enforce that agents can only push to designated branches (e.g., `agent/*` or `copilot/*`), that all agent-created PRs require independent review (human or evaluator agent), and that CI/CD checks must pass before merge, so that agents cannot ship code directly to protected branches.

**8.11** — As an engineer, I want all agent-produced commits to be co-authored — attributing both the agent and the mission that produced them — so that every line of agent-written code is traceable to its origin and the human who authorized the mission.

**8.12** — As an engineer, I want the harness to detect when an agent's behavior deviates significantly from its assigned persona, scope, or expected action patterns — using behavioral signals (e.g., unexpected file access patterns, unusual tool call sequences) as a heuristic gate, invoking LLM analysis only when signals flag an anomaly — and halt execution with an alert if confirmed, so that scope violations are caught even if the intermediary would technically permit the individual actions. *(Applies P.4)*

**8.13** — As an engineer, I want the harness to enforce memory isolation between missions, between agents, and between users, so that sensitive context from one mission cannot leak into another, and a compromised agent session cannot access state belonging to other missions or users.

---

## Section 9: Observability & Debugging

**9.1** — As an engineer, I want the harness to instrument every agent interaction from day one — model invocations, tool calls, reasoning steps, intermediary decisions, context assembly — using OpenTelemetry traces, so that I can drill into any step of any mission without having set up custom logging after the fact. *(Applies P.1)*

**9.2** — As an engineer, I want a local observability stack (logs, metrics, traces) that is ephemeral and scoped per mission, spun up automatically when a mission starts and torn down when it completes, so that agents can query their own logs and metrics during execution and I get clean, isolated observability per mission without managing shared infrastructure. *(Applies P.1)*

**9.3** — As an engineer, I want agents to be able to query their own logs (via LogQL or equivalent) and metrics (via PromQL or equivalent) as part of their workflow — enabling prompts like "ensure service startup completes in under 800ms" — so that agents can self-diagnose performance issues and validate non-functional requirements without me interpreting logs for them.

**9.4** — As an engineer, I want the harness to compute lightweight behavioral signals in real time — directly from agent traces without LLM involvement — covering categories like repetition/looping (bigram similarity), tool failure frequency, retry escalation patterns, context thrashing (frequent compaction/resets), efficiency scores (turns vs. baseline expectations), and task stalling (no meaningful progress over N steps), so that the harness has a cheap, fast first-pass assessment of agent health at all times. *(Defines P.4)*

**9.5** — As an engineer, I want behavioral signals to act as gates that determine what happens next: signals within normal range require no further action, signals crossing configurable warning thresholds trigger LLM-based evaluation to diagnose the issue, and signals crossing critical thresholds halt agent execution and alert me immediately — so that expensive LLM evaluation is reserved for situations that actually warrant it, and clear failures are caught instantly without waiting for an LLM to weigh in. *(Defines P.4, applies P.1)*

**9.6** — As an engineer, I want to define custom signal types beyond the defaults — domain-specific heuristics like "agent touched files outside its declared blast radius," "agent generated code with no corresponding test," or "agent skipped a required verification step" — so that the signal system reflects my project's specific quality concerns, not just generic behavioral patterns.

**9.7** — As an engineer, I want all signals to be emitted as OpenTelemetry trace attributes (e.g., `signals.quality`, `signals.repetition.count`, `signals.efficiency_score`, `signals.tool_failure.rate`) and flagged visually in trace UIs when concerning, so that I can query, filter, dashboard, and alert on signals using my existing observability platform.

**9.8** — As an engineer, I want the harness to aggregate signals into an overall mission health assessment (e.g., Excellent / Good / Neutral / Poor / Severe) that updates continuously as the mission runs, so that I can see at a glance whether a mission needs attention without inspecting individual signals. *(Applies P.1)*

**9.9** — As an engineer, I want the harness to expose a real-time dashboard per active mission showing agent status, current task, context utilization, token spend, wall-clock time, signal health, and intermediary decision log, so that I can monitor running missions at a glance without reading raw trace data.

**9.10** — As an engineer, I want the harness to wire browser DevTools Protocol (CDP) into the agent runtime, giving agents the ability to take DOM snapshots, capture screenshots, navigate pages, reproduce bugs, and validate fixes in a running application instance, so that agents can debug UI issues the same way I would.

**9.11** — As an engineer, I want the harness to boot a separate application instance per mission (per git worktree), so that agents can launch, drive, and inspect their own instance without interfering with other running missions or my local development environment.

**9.12** — As an engineer, I want the harness to capture structured error context when any step fails — the agent's state at failure, the context window contents, the tool call that failed, the error returned, and what the agent attempted next — so that debugging a failed mission is forensic analysis of a clear record, not guesswork from partial logs.

**9.13** — As an engineer, I want the harness to detect recurring failure patterns across missions — using signal trends, not just per-mission snapshots — and surface them as harness improvement recommendations (e.g., "tool X has a 40% failure rate across the last 10 missions — consider replacing or fixing it"), so that systemic issues become visible rather than hiding in per-mission noise. *(Applies P.4)*

**9.14** — As an engineer, I want the harness to provide a full trace replay for any completed or failed mission — stepping through every agent decision, context state, tool call, signal value, and intermediary action in sequence — so that I can review exactly what happened and why, whether for debugging, auditing, or calibrating the harness.

**9.15** — As an engineer, I want the harness to track token usage breakdowns per mission — tokens spent on context assembly vs. reasoning vs. tool outputs vs. compaction overhead vs. evaluation — so that I can identify where token budget is being wasted and optimize cost without guessing.

**9.16** — As an engineer, I want the harness to create a reinforcement loop: signals identify problematic or exemplary missions, I review the flagged traces and apply fixes (prompt updates, tool changes, harness config), and the harness monitors signal metrics after redeployment to validate that improvements took effect, so that observability directly drives continuous harness improvement.

---

## Section 10: Architectural Enforcement & Entropy Management

**10.1** — As an engineer, I want to define architectural invariants (e.g., "parse data shapes at the boundary," "no direct database access from UI layer," "all API endpoints require authentication") and have the harness enforce them mechanically via linters, CI checks, and structural tests — not by instructing agents in prose — so that constraints apply uniformly to every line of code, whether written by an agent or a human.

**10.2** — As an engineer, I want to define a strict layered architecture (e.g., Types → Config → Repo → Service → Runtime → UI) with explicit rules about which layers can depend on which, enforced at CI level, so that agents cannot introduce cross-layer violations no matter how they choose to implement a solution.

**10.3** — As an engineer, I want to encode my domain expertise and taste — React patterns, error handling conventions, security practices, naming standards — as mechanical rules once, and have those rules enforced continuously across every agent-produced change, so that "human taste is captured once, then enforced everywhere" rather than re-explained per mission.

**10.4** — As an engineer, I want custom lint error messages to include remediation instructions that are injected directly into agent context when violations are detected, so that the tooling teaches the agent how to fix the issue in the moment rather than just flagging a failure.

**10.5** — As an engineer, I want the harness to enforce invariants, not implementations — specifying *what* must be true (e.g., "all inputs are validated at the boundary") but not *how* the agent achieves it — so that agents retain autonomy in their approach while architectural guarantees hold. *(Applies P.6 — autonomy within constraints)*

**10.6** — As an engineer, I want the harness to run background entropy management agents on a configurable cadence (sensible default provided) that scan the codebase for deviations from established patterns, quality drift, and inconsistencies, and open targeted refactoring PRs — with each PR small enough to review in under a minute and auto-merged if CI passes — so that cleanup is continuous and automatic rather than piling up as a Friday chore. *(Applies P.1)*

**10.7** — As an engineer, I want the harness to run a dedicated doc-gardening agent that scans for stale, obsolete, or broken documentation — dead links, outdated API references, instructions that no longer match the code — and opens fix-up PRs, so that the knowledge base agents depend on stays accurate without me manually auditing docs.

**10.8** — As an engineer, I want entropy management agents and doc-gardening agents to be gated by behavioral signals and heuristics — scanning cheaply with pattern matching and structural analysis first, only invoking LLM-based analysis when heuristics flag a likely issue — so that background maintenance doesn't become a runaway token cost. *(Applies P.4)*

**10.9** — As an engineer, I want the harness to maintain a quality scoring document per repository that grades each product domain and architectural layer, updated automatically by entropy management agents, so that I can track quality trends over time and see where drift is accumulating before it becomes a problem.

**10.10** — As an engineer, I want the harness to detect when agents replicate existing patterns — even suboptimal or inconsistent ones — and flag the replication against the declared invariants, so that bad patterns don't silently propagate just because they already exist in the codebase.

**10.11** — As an engineer, I want CI to validate that the harness knowledge base (AGENTS.md, docs/ directory, decision records) is up to date, cross-linked, and structurally correct on every commit, so that knowledge rot is caught as a build failure rather than discovered when an agent makes a wrong decision based on stale information.

**10.12** — As an engineer, I want to record architectural decisions as decision records (ADRs) in the repository, and have the harness surface relevant ADRs to agents when they're working in areas affected by those decisions, so that agents understand *why* constraints exist, not just *what* the constraints are.

**10.13** — As an engineer, I want the harness to track a "technical debt" metric per area of the codebase — derived from entropy management scan results, lint violation density, test coverage, and signal trends — and prioritize cleanup PRs toward high-debt areas automatically, so that debt is paid down continuously in small increments rather than compounding. *(Applies P.1)*

**10.14** — As an engineer, I want to be able to override or waive specific invariants for a particular mission (with an explicit justification recorded in the mission's sprint contract), so that the enforcement system doesn't block legitimate exceptions, while ensuring every exception is documented and traceable.

---

## Section 11: Cost Monitoring & Optimization

**11.1** — As an engineer, I want the harness to track token usage in real time per mission — broken down by agent role (planner, generator, evaluator, sub-agents), by purpose (context assembly, reasoning, tool calls, compaction, evaluation), and by model — so that I know exactly where tokens are being spent and can identify waste without post-hoc analysis.

**11.2** — As an engineer, I want the harness to ship with sensible default token budgets and time limits per mission, sprint, and agent role — derived from mission complexity, model pricing, and historical baselines — that work out of the box without any configuration, while allowing me to override any default when I need tighter or looser limits for a specific repo or mission. *(Applies P.1)*

**11.3** — As an engineer, I want the harness to ship with a default model selection ladder that assigns the most capable model to planning and evaluation roles and cost-effective models to sub-agent and routine tasks — working out of the box — while allowing me to override assignments per role, per mission type, or per repo, and automatically benchmarking whether quality holds when cheaper models are used. *(Applies P.1)*

**11.4** — As an engineer, I want the harness to use sub-agents as context firewalls for token efficiency — ensuring the parent agent sees only the prompt and condensed result (1,000–2,000 tokens), never the sub-agent's full intermediate reasoning — so that token accumulation from tool-heavy subtasks doesn't blow up the orchestrator's budget.

**11.5** — As an engineer, I want the harness to cache subtask plans and reuse them for similar future missions — storing plans in a searchable index keyed by task description similarity — so that the harness avoids redundant LLM calls for decomposition and planning work it has already done.

**11.6** — As an engineer, I want the harness to summarize verbose tool outputs before injecting them into context, stripping metadata and returning only fields relevant to the agent's current task, so that tool results don't consume disproportionate context window space.

**11.7** — As an engineer, I want the harness to apply the signal-gated evaluation model — running cheap heuristic checks always and invoking expensive LLM-based evaluation only when signals warrant it — so that evaluation cost scales with actual risk rather than being a flat tax on every step. *(Applies P.4)*

**11.8** — As an engineer, I want the harness to produce a cost report per completed mission — total token spend, cost in dollars, breakdown by agent and purpose, comparison to budget, and comparison to similar past missions — so that I can review cost efficiency and spot trends over time.

**11.9** — As an engineer, I want the harness to flag missions that significantly exceed the cost profile of similar past missions — using historical baselines, not just absolute thresholds — so that cost anomalies are caught in context rather than only when a hard budget limit is hit. *(Applies P.4 — heuristic-first detection)*

**11.10** — As an engineer, I want the harness to recommend cost optimizations based on actual usage data — e.g., "evaluator consumed 40% of tokens on this mission but only caught 2 issues; consider lightweight evaluation for this mission type," or "sub-agent X used 15K tokens and returned a 200-token summary with no issues found; consider skipping this delegation" — so that optimization is data-driven rather than speculative. *(Applies P.5)*

**11.11** — As an engineer, I want the harness to automatically calibrate cost measures to mission stakes by default — applying lighter evaluation and cheaper models for low-complexity missions, full-cost measures for high-complexity missions — inferred from mission scope and history without manual configuration, while allowing me to override the classification for any mission. *(Applies P.1)*

**11.12** — As an engineer, I want the harness to track cumulative cost across all missions over configurable time windows (daily, weekly, monthly) and alert me when spend approaches or exceeds budget thresholds I define, so that I maintain financial control at the portfolio level, not just per mission.

**11.13** — As an engineer, I want every cost-related configuration (budgets, limits, model assignments, evaluation intensity, alerting thresholds) to ship with sensible defaults that work out of the box, derived from mission complexity, model pricing, and accumulated usage data — so that the harness is cost-aware from the first mission without requiring me to set up cost policies before I can start working. All defaults must be inspectable and overridable at the repo, mission type, or individual mission level. *(Defines P.1 for cost domain)*

---

## Section 12: Harness Lifecycle & Evolution

**12.1** — As an engineer, I want the harness to treat every component as an encoded assumption about what the current model can't do, and surface those assumptions explicitly — listing what each harness component compensates for and what model capability it assumes is missing — so that I can see at a glance which parts of the harness exist because of model limitations versus which exist for architectural reasons that won't go away.

**12.2** — As an engineer, I want the harness to prompt me to re-evaluate its components when a new model is released — automatically running the existing evaluation suite with the new model against a minimal harness configuration and comparing results to the current full-harness baseline — so that I have data on which components are still load-bearing and which can be removed. *(Applies P.5)*

**12.3** — As an engineer, I want to simplify the harness methodically — removing one component at a time and reviewing the impact on evaluation scores and signal health — rather than making radical cuts, so that I can tell which pieces were actually load-bearing and which were dead weight.

**12.4** — As an engineer, I want the harness to track which components were removed, when, why, and what the measured impact was, stored as a lifecycle decision record in the repo, so that simplification decisions are documented and reversible if a future model regresses in a relevant capability.

**12.5** — As an engineer, I want the harness to detect when a component that was previously necessary is now redundant — by monitoring whether it ever triggers, changes an outcome, or catches an error that wouldn't have been caught otherwise — and recommend its removal with supporting data, so that harness complexity decreases naturally over time rather than only when I remember to audit it. *(Applies P.4 — signal-based detection)*

**12.6** — As an engineer, I want the harness to also detect when new model capabilities create opportunities to add new components — e.g., a model that's now reliable enough at self-evaluation might allow a lightweight self-check before the full evaluator pass — so that the harness evolves in both directions: simpler where models have improved, richer where new capabilities unlock new patterns.

**12.7** — As an engineer, I want the harness to version its own configuration alongside the codebase — so that any commit can be reproduced with the exact harness configuration that produced it, and I can diff harness configurations across time to understand how the system has evolved.

**12.8** — As an engineer, I want the harness to maintain a changelog of its own evolution — what was added, removed, or modified, what model or evaluation data motivated the change, and what the measured impact was — so that the harness's history is as well-documented as the codebase it manages.

**12.9** — As an engineer, I want the harness to pass the future-proofing test: if I swap in a more powerful model without changing the harness, mission quality should improve, not degrade — and the harness should verify this automatically when a new model is configured, so that I have confidence the harness isn't overfitted to a specific model's quirks.

**12.10** — As an engineer, I want the harness to support A/B comparison of harness configurations — running the same mission with two different configurations and comparing evaluation scores, signal health, cost, and latency side by side — so that I can make data-driven decisions about harness changes rather than guessing.

**12.11** — As an engineer, I want the defaults the harness ships with to improve over time based on accumulated data from my repos and missions — adjusting baseline budgets, model assignments, evaluation intensity, and signal thresholds to reflect actual usage patterns — while never overriding a default I've explicitly set, so that the harness gets better at working out of the box the more I use it. *(Applies P.1)*

**12.12** — As an engineer, I want the harness to surface a periodic "harness health report" — summarizing component utilization, redundancy candidates, default drift from actuals, cost trends, and signal baselines — so that harness maintenance is a lightweight, data-informed activity rather than an ad-hoc audit I have to remember to do.

---

## Summary

### Story count by section

| Section | Stories |
|---------|---------|
| Cross-Cutting Principles | 6 |
| 1. Project Bootstrapping & Configuration | 9 |
| 2. Mission Definition & Delegation | 10 |
| 3. Context Management | 11 |
| 4. Tool & Integration Management | 14 |
| 5. Memory & State Persistence | 15 |
| 6. Evaluation & Quality Assurance | 13 |
| 7. Multi-Agent Orchestration | 12 |
| 8. Security & Trust Boundaries | 13 |
| 9. Observability & Debugging | 16 |
| 10. Architectural Enforcement & Entropy Management | 14 |
| 11. Cost Monitoring & Optimization | 13 |
| 12. Harness Lifecycle & Evolution | 12 |
| **Total** | **158** |

### Cross-cutting principle coverage

| Principle | Sections where it applies |
|-----------|--------------------------|
| P.1 Zero-Config Sensible Defaults | 1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12 |
| P.2 Originating Platform Communication | 2, 5, 6, 7, 8 |
| P.3 Deterministic Intermediary | 2, 4, 5, 6, 7, 8 |
| P.4 Signal-Gated Evaluation | 6, 8, 9, 10, 11, 12 |
| P.5 Automation-First with Human Approval | 1, 2, 4, 5, 11, 12 |
| P.6 Two-Zone Security Model | 4, 5, 7, 8, 10 |
