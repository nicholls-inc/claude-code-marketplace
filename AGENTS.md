# AGENTS.md

Cross-runtime entry point for any agent (Claude Code, Codex, Cursor, Copilot, Devin, Windsurf, Gemini CLI) working in this repo.

## Read before non-trivial changes

Before any non-trivial change, walk up from the file or directory you're touching to the repo root and read every `JOURNAL.md` you pass. Newest entries are at the top. The journals carry the narrative record of why things look the way they do — what was tried, what was rolled back, what's still open.

Trivial edits (typo fixes, single-line changes with obvious intent) don't need the walk. If you're unsure, read.

## Marketplace structure

For plugin layout, build commands, commit conventions, and Dafny/Lean limits, see [CLAUDE.md](CLAUDE.md). This file is about navigation; CLAUDE.md is about contents.
