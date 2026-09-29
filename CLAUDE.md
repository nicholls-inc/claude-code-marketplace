# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server. Some plugins live in their own repositories, and `.claude-plugin/marketplace.json` points at them.

## Plugins

### crosscheck (`nicholls-inc/crosscheck`)

Crosschecks Claude's code claims using Dafny formal verification for provably correct Python/Go code, plus semi-formal reasoning for structured code analysis. The plugin lives in `crosscheck/` of [nicholls-inc/crosscheck](https://github.com/nicholls-inc/crosscheck), and the marketplace installs it from there with a `git-subdir` source. Develop and file issues there.

### awesome-copilot (`awesome-copilot/`)

Meta prompts for discovering and installing curated GitHub Copilot customizations.

- **Skills** (`awesome-copilot/skills/`): `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills`
- **Agent** (`awesome-copilot/agents/project-scaffold.md`): End-to-end project scaffolding

### field-report (`field-report/`)

Field report plugin. Generates structured performance reports on plugins, skills, and agents by analysing Claude Code session conversations.

- **Skills** (`field-report/skills/`): `/field-report`

### pr-swarm (`pr-swarm/`)

Multi-agent PR review swarm and review-triage plugin for automated, multi-lens code reviews and feedback triage.

- **Skills** (`pr-swarm/skills/`):
  - `/pr-swarm`: Router-first PR review swarm coordinating specialized lenses (`security`, `test-theatre`, `xp`, `crosscheck/byfuglien`, `crosscheck/hellebuyck`), cross-lens deduplication, structured inline comments, and sticky summary upsert.
  - `/review-triage`: Automated feedback triage acting on review comments within strict autonomy boundaries, human-participation gating, automated commit application with loop-prevention trailers, and auto-merge preconditioning.

## Tools

### claude-github-app (`tools/claude-github-app/`)

Local Go wrapper that intercepts `claude` invocations and injects a GitHub App installation token chosen by working directory. Not a Claude plugin — a developer tool that lives under `tools/`. Installs to `~/bin/claude` and `~/bin/claude-github-app`, shadowing the real `claude` on PATH and execing it with isolated `GH_CONFIG_DIR` and `GIT_CONFIG_GLOBAL`. First Go module in this repo; uses `github.com/BurntSushi/toml` and `github.com/golang-jwt/jwt/v5`.

Optional `gh` + `git` PATH shims (`make install-shims`) solve mid-session token expiry: every `gh`/`git` invocation re-reads the shared `~/.cache/claude-github-app/` cache and re-mints if within the 5-minute refresh window. Unmapped CWDs pass through to real `gh`/`git` unmodified. Shim runtime is in `internal/shim/`; per-tool injection logic in `cmd/gh/main.go` and `cmd/git/main.go`.

- Build: `cd tools/claude-github-app && make build`
- Install: `make install` (claude + claude-github-app) or `make install-all` (also installs gh + git shims)
- Test: `make test` (pure Go, no Docker)

## Commit conventions

Conventional commits enforced via commitlint + husky.

**Behavioral artifacts** (`SKILL.md`, `agents/*.md`) define agent/skill behavior and are functional code. Commits that touch them must use a release-triggering type:

- `feat(<plugin>):` — new or expanded behavior (minor bump)
- `fix(<plugin>):` — corrective behavior change (patch bump)

`docs:` and `refactor:` are **both blocked** on behavioral artifacts (enforced by `.husky/commit-msg`). release-please treats `refactor:` as non-user-facing, so behavior changes filed as `refactor:` will silently stall the release pipeline. If a change to `SKILL.md` or `agents/*.md` is genuinely non-behavioral (rare — usually internal renames or comment-only edits), split it into a separate commit that does not touch a behavioral artifact.

- `feat(field-report): add new analysis dimension` — new skill behavior
- `fix(pr-swarm): correct lens routing threshold` — bug fix in skill logic
- `docs(pr-swarm): update README installation steps` — actual documentation (not a behavioral artifact)
