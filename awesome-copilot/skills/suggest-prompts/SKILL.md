# /suggest-prompts — Suggest Awesome GitHub Copilot Prompts

## Description

Analyze current repository context and suggest relevant prompt files from the [awesome-copilot repository](https://github.com/github/awesome-copilot/blob/main/docs/README.prompts.md) that are not already installed locally. Identifies outdated prompts needing updates and presents recommendations for user approval before any installation.

## Instructions

You are a discovery assistant for GitHub Copilot prompt files. Your job is to fetch the curated list of prompts from awesome-copilot, compare against locally installed prompts, and present actionable recommendations.

### Step 1: Fetch Available Prompts

Extract the prompts list and descriptions from the awesome-copilot documentation.

- Use `WebFetch` to retrieve `https://github.com/github/awesome-copilot/blob/main/docs/README.prompts.md`
- Parse the prompt names, filenames, and descriptions from the document

### Step 2: Scan Local Prompts

Discover existing prompt files already installed in the repository.

- Use `Glob` with pattern `.github/prompts/*.prompt.md` to find local prompt files
- Use `Read` on each discovered file to extract front matter descriptions

### Step 3: Fetch Remote Versions for Comparison

For each locally installed prompt, fetch the corresponding remote version to check for updates.

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/prompts/<filename>`
- Compare entire file content (front matter and body)
- Identify specific differences: front matter changes (description, tools, mode), tools array modifications, content updates

### Step 4: Analyze Repository Context

Review the repository to understand what prompts would be most useful.

**Repository Patterns to check:**
- Use `Glob` and `Read` to identify programming languages (.cs, .js, .py, etc.)
- Framework indicators (ASP.NET, React, Azure, etc.)
- Project types (web apps, APIs, libraries, tools)
- Documentation needs (README, specs, ADRs)

### Step 5: Present Recommendations

Display analysis results in a structured table:

| Awesome-Copilot Prompt | Description | Already Installed | Similar Local Prompt | Suggestion Rationale |
|-|-|-|-|-|
| [prompt-name.prompt.md](https://github.com/github/awesome-copilot/blob/main/prompts/prompt-name.prompt.md) | Description | Status | Local match | Rationale |

**Status icons:**
- ✅ Already installed and up-to-date
- ⚠️ Installed but outdated (update available)
- ❌ Not installed in repo

**AWAIT** user request to proceed with installation or updates. DO NOT INSTALL OR UPDATE UNLESS DIRECTED TO DO SO.

### Step 6: Download or Update Requested Prompts

When the user approves specific prompts:

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/prompts/<filename>`
- Save new prompts to `.github/prompts/` using `Write`
- For updates, replace the local file with the remote version
- Do NOT adjust content of the downloaded files — copy them as-is

## Arguments

No arguments required. The skill automatically analyzes the current repository context.

Example: `/suggest-prompts`
