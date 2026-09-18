# Intent: Port and integrate pr-swarm plugin into claude-code-marketplace

## Problem statement
The marketplace provides plugins for formal verification (`crosscheck`), copilot discovery (`awesome-copilot`), session reporting (`field-report`), and issue scheduling (`xylem`), but lacks an automated, multi-lens pull request review and triage capability. Today, pull request reviews in Claude Code either require manual prompt coordination or single-agent passes that overlook specialized risk lenses such as security, test theatre, extreme programming design principles, and specification/invariant correctness. Furthermore, there is no automated triage loop to safely address review feedback, apply fixes, and prepare PRs for merge without violating human reviewer sovereignty.

## Proposed outcome
The `pr-swarm` plugin is ported from upstream into this marketplace as a project-agnostic, standalone plugin at `./pr-swarm`. Once integrated:
1. Users can install `pr-swarm@nicholls` directly via Claude Code plugin commands.
2. Two cooperative skills become available:
   - `/pr-swarm`: Orchestrates a router-first review pipeline dispatching specialized lenses (`security`, `test-theatre`, `xp`, `crosscheck/byfuglien`, `crosscheck/hellebuyck`), cross-lens deduplication, non-approving inline comments (`event=COMMENT`), and idempotent sticky summaries.
   - `/review-triage`: Automates the resolution of review feedback within an autonomy ladder, enforces human-participation boundaries (INV-3), attaches loop-prevention git trailers (`PR-Swarm-Automated: true`), and validates auto-merge safety preconditions.
3. The plugin is registered in `.claude-plugin/marketplace.json`, automated releases are managed via `release-please-config.json` and `.release-please-manifest.json` starting at baseline version `0.2.0`, and documented in repository guides (`README.md` and `CLAUDE.md`).
4. All organizational specifics (`ev.energy`, `ev-energy`, internal PR fixtures) are scrubbed in favor of clean project-agnostic defaults.

## Affected users and systems
- **Plugin Users & Developers**: Can run automated multi-lens PR reviews and triage workflows across GitHub repositories using `/pr-swarm` and `/review-triage`.
- **Marketplace Catalog**: `.claude-plugin/marketplace.json` gains the `pr-swarm` plugin definition.
- **Release Automation**: Release-please monitors `pr-swarm` path changes and manages version bumps and changelog releases.
- **Crosscheck Interoperability**: `pr-swarm` leverages the marketplace's `crosscheck` plugin for `crosscheck/byfuglien` and `crosscheck/hellebuyck` formal verification and specification lenses.

## Constraints
- **Zero Company Leaks**: No internal or organization-specific strings (`ev.energy`, `ev-energy`, internal repo references, or specific company PR IDs).
- **Core Review & Triage Invariants**:
  - Never Approve: Automated reviews must never submit `event=APPROVE`. Human approvals remain the sovereign gate.
  - Human Participation Gate (INV-3): Review threads with human participation must never be resolved or answered by automated triage without explicit human direction.
  - Loop Prevention: All automated triage commits must carry `PR-Swarm-Automated: true`.
  - Single Sticky Summary: PR summary comments must be updated idempotently using `<!-- pr-swarm-summary -->`.
- **Governance & Change Tier**: Tier 2 standard change requiring committed `intent/` and root `spec.md`, without touching protected surfaces.

## Open questions
None — plugin structure, skills, and configuration follow existing marketplace conventions and are fully defined.
