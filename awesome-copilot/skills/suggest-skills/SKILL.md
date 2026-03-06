# /suggest-skills — Suggest Awesome GitHub Copilot Skills

## Description

Analyze current repository context and suggest relevant Agent Skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot/blob/main/docs/README.skills.md) that are not already installed locally. Skills are self-contained folders containing a `SKILL.md` and optional bundled assets. Identifies outdated skills needing updates and presents recommendations for user approval before any installation.

## Instructions

You are a discovery assistant for GitHub Copilot agent skills. Your job is to fetch the curated list of skills from awesome-copilot, compare against locally installed skills, and present actionable recommendations.

### Step 1: Fetch Available Skills

Extract the skills list and descriptions from the awesome-copilot documentation.

- Use `WebFetch` to retrieve `https://github.com/github/awesome-copilot/blob/main/docs/README.skills.md`
- Parse the skill names, folder names, and descriptions from the document

### Step 2: Scan Local Skills

Discover existing skill folders already installed in the repository.

- Use `Glob` with pattern `.github/skills/*/SKILL.md` to find local skill files
- Use `Read` on each discovered `SKILL.md` to extract front matter (`name`, `description`)
- Use `Glob` to list any bundled assets within each skill folder

### Step 3: Fetch Remote Versions for Comparison

For each locally installed skill, fetch the corresponding remote version to check for updates.

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/skills/<skill-name>/SKILL.md`
- Compare entire file content (front matter and body)
- Identify specific differences: front matter changes, instruction updates, bundled asset changes

### Step 4: Analyze Repository Context

Review the repository to understand what skills would be most useful.

**Repository Patterns to check:**
- Use `Glob` and `Read` to identify programming languages (.cs, .js, .py, .ts, etc.)
- Framework indicators (ASP.NET, React, Azure, Next.js, etc.)
- Project types (web apps, APIs, libraries, tools, infrastructure)
- Development workflow requirements (testing, CI/CD, deployment)
- Infrastructure and cloud providers (Azure, AWS, GCP)

### Step 5: Present Recommendations

Display analysis results in a structured table:

| Awesome-Copilot Skill | Description | Bundled Assets | Already Installed | Similar Local Skill | Suggestion Rationale |
|-|-|-|-|-|-|
| [skill-name](https://github.com/github/awesome-copilot/tree/main/skills/skill-name) | Description | Asset count | Status | Local match | Rationale |

**Status icons:**
- ✅ Already installed and up-to-date
- ⚠️ Installed but outdated (update available)
- ❌ Not installed in repo

**AWAIT** user request to proceed with installation or updates. DO NOT INSTALL OR UPDATE UNLESS DIRECTED TO DO SO.

### Step 6: Download or Update Requested Skills

When the user approves specific skills:

- Create the skill folder under `.github/skills/<skill-name>/` using `Bash` with `mkdir -p`
- Use `Bash` with `curl` to download `SKILL.md` from `https://raw.githubusercontent.com/github/awesome-copilot/main/skills/<skill-name>/SKILL.md`
- Save using `Write` to `.github/skills/<skill-name>/SKILL.md`
- Also download any bundled assets (scripts, templates, data files) listed in the remote skill folder
- For updates, replace the entire local skill folder with the remote version
- Do NOT adjust content of the downloaded files — copy them as-is

## Arguments

No arguments required. The skill automatically analyzes the current repository context.

Example: `/suggest-skills`
