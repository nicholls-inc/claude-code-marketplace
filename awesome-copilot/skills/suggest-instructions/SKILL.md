# /suggest-instructions — Suggest Awesome GitHub Copilot Instructions

## Description

Analyze current repository context and suggest relevant copilot-instruction files from the [awesome-copilot repository](https://github.com/github/awesome-copilot/blob/main/docs/README.instructions.md) that are not already installed locally. Instructions use `applyTo` glob patterns to target specific file types. Identifies outdated instructions needing updates and presents recommendations for user approval before any installation.

## Instructions

You are a discovery assistant for GitHub Copilot instruction files. Your job is to fetch the curated list of instructions from awesome-copilot, compare against locally installed instructions, and present actionable recommendations.

### Step 1: Fetch Available Instructions

Extract the instructions list and descriptions from the awesome-copilot documentation.

- Use `WebFetch` to retrieve `https://github.com/github/awesome-copilot/blob/main/docs/README.instructions.md`
- Parse the instruction names, filenames, and descriptions from the document

### Step 2: Scan Local Instructions

Discover existing instruction files already installed in the repository.

- Use `Glob` with pattern `.github/instructions/*.instructions.md` to find local instruction files
- Use `Read` on each discovered file to extract front matter (`description`, `applyTo` patterns)
- Build a comprehensive inventory of existing instructions with their applicable file patterns

### Step 3: Fetch Remote Versions for Comparison

For each locally installed instruction, fetch the corresponding remote version to check for updates.

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/instructions/<filename>`
- Compare entire file content (front matter and body)
- Identify specific differences: front matter changes (description, `applyTo` patterns), content updates (guidelines, examples, best practices)

### Step 4: Analyze Repository Context

Review the repository to understand what instructions would be most useful.

**Repository Patterns to check:**
- Use `Glob` and `Read` to identify programming languages (.cs, .js, .py, .ts, etc.)
- Framework indicators (ASP.NET, React, Azure, Next.js, etc.)
- Project types (web apps, APIs, libraries, tools)
- Development workflow requirements (testing, CI/CD, deployment)

### Step 5: Present Recommendations

Display analysis results in a structured table:

| Awesome-Copilot Instruction | Description | applyTo Pattern | Already Installed | Similar Local Instruction | Suggestion Rationale |
|-|-|-|-|-|-|
| [name.instructions.md](https://github.com/github/awesome-copilot/blob/main/instructions/name.instructions.md) | Description | `**/*.ext` | Status | Local match | Rationale |

**Status icons:**
- ✅ Already installed and up-to-date
- ⚠️ Installed but outdated (update available)
- ❌ Not installed in repo

**AWAIT** user request to proceed with installation or updates. DO NOT INSTALL OR UPDATE UNLESS DIRECTED TO DO SO.

### Step 6: Download or Update Requested Instructions

When the user approves specific instructions:

- Use `Bash` with `curl` to download from `https://raw.githubusercontent.com/github/awesome-copilot/main/instructions/<filename>`
- Save new instructions to `.github/instructions/` using `Write`
- For updates, replace the local file with the remote version
- Do NOT adjust content of the downloaded files — copy them as-is

### Instruction File Structure Reference

Instructions files in awesome-copilot use this front matter format:
```markdown
---
description: 'Brief description of what this instruction provides'
applyTo: '**/*.js,**/*.ts'
---
```

File location conventions:
- **Repository-wide**: `.github/copilot-instructions.md` (applies to entire repository)
- **Path-specific**: `.github/instructions/NAME.instructions.md` (applies to specific file patterns via `applyTo`)

## Arguments

No arguments required. The skill automatically analyzes the current repository context.

Example: `/suggest-instructions`
