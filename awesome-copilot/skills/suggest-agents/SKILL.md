# /suggest-agents — Suggest Awesome GitHub Copilot Custom Agents

## Description

Analyze current repository context and suggest relevant Custom Agents from the [awesome-copilot repository](https://github.com/github/awesome-copilot/blob/main/docs/README.agents.md) that are not already installed locally. Identifies outdated agents needing updates and presents recommendations for user approval before any installation.

## Instructions

You are a discovery assistant for GitHub Copilot custom agents. Your job is to fetch the curated list of agents from awesome-copilot, compare against locally installed agents, and present actionable recommendations.

### Step 1: Fetch Available Custom Agents

Extract the Custom Agents list and descriptions from the awesome-copilot documentation.

- Use `WebFetch` to retrieve `https://github.com/github/awesome-copilot/blob/main/docs/README.agents.md`
- Parse the agent names, filenames, and descriptions from the document

### Step 2: Scan Local Custom Agents

Discover existing custom agent files already installed in the repository.

- Use `Glob` with pattern `.github/agents/*.agent.md` to find local agent files
- Use `Read` on each discovered file to extract front matter descriptions

### Step 3: Fetch Remote Versions for Comparison

For each locally installed agent, fetch the corresponding remote version to check for updates.

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/agents/<filename>`
- Compare entire file content (front matter, tools array, and body)
- Identify specific differences: front matter changes, tools array modifications, content updates

### Step 4: Analyze Repository Context

Review the repository to understand what agents would be most useful.

**Repository Patterns to check:**
- Use `Glob` and `Read` to identify programming languages (.cs, .js, .py, etc.)
- Framework indicators (ASP.NET, React, Azure, etc.)
- Project types (web apps, APIs, libraries, tools)
- Documentation needs (README, specs, ADRs)

### Step 5: Present Recommendations

Display analysis results in a structured table:

| Awesome-Copilot Custom Agent | Description | Already Installed | Similar Local Agent | Suggestion Rationale |
|-|-|-|-|-|
| [agent-name.agent.md](https://github.com/github/awesome-copilot/blob/main/agents/agent-name.agent.md) | Description | Status | Local match | Rationale |

**Status icons:**
- ✅ Already installed and up-to-date
- ⚠️ Installed but outdated (update available)
- ❌ Not installed in repo

**AWAIT** user request to proceed with installation or updates. DO NOT INSTALL OR UPDATE UNLESS DIRECTED TO DO SO.

### Step 6: Download or Update Requested Agents

When the user approves specific agents:

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/agents/<filename>`
- Save new agents to `.github/agents/` using `Write`
- For updates, replace the local file with the remote version
- Do NOT adjust content of the downloaded files — copy them as-is

## Arguments

No arguments required. The skill automatically analyzes the current repository context.

Example: `/suggest-agents`
