---
name: implement-feature
description: >-
  Implement a low-effort feature from a GitHub issue URL, then open a PR.
  Use when given a GitHub issue URL for a feature that has been refined
  and labeled as ready for autonomous implementation.
argument-hint: <github-issue-url>
disable-model-invocation: true
allowed-tools: Bash, Read, Edit, Write, Grep, Glob
---

# Implement Feature from GitHub Issue

You are an autonomous feature-implementation agent running in a non-interactive session. Your job is to implement the feature described in the given GitHub issue, then open a PR.

**Input**: $ARGUMENTS — a GitHub issue URL (e.g., `https://github.com/owner/repo/issues/42`)

You are already running in a git worktree branched from the default branch. Do NOT create a new branch. Work on the current branch.

## Phase 1: Parse & Setup

1. Extract the issue number from the URL.
2. Fetch full issue details (title, body, comments, labels):
   ```
   gh issue view <number> --json title,body,comments,labels,url
   ```
3. Verify the issue is open.
4. Read the issue thoroughly — title, body, and all comments.

## Phase 2: Plan Implementation

5. Identify:
   - What feature is being requested
   - What acceptance criteria exist (explicit or implied)
   - Which files/modules are likely affected

6. Search the codebase for related patterns:
   - Use `Grep` to find similar features or related code
   - Read relevant source files to understand existing patterns

7. Create an implementation plan (in your working memory):
   - Which files to create or modify
   - What patterns to follow from existing code
   - What tests to add

8. **SCOPE CHECK** — Count the files your plan requires changing:
   - If more than **10 files** need to be created or modified: **STOP**
     - Run: `gh issue comment <number> --body "This feature appears to require changes to more than 10 files, which exceeds the scope for autonomous implementation. Flagging for human review."`
     - Exit without making any code changes
   - If 10 files or fewer: proceed to Phase 3

## Phase 3: Implement

9. Follow the plan from Phase 2.
10. Follow existing code patterns discovered in Phase 2.
11. Implement the minimum viable version — no gold-plating, no bonus features.
12. Add tests using the project's existing test patterns.
13. Discover the test command (check in order):
    - Read `CLAUDE.md` — look for a "test" or "run tests" section
    - Check `package.json` for a `test` script
    - Check `Makefile` for a `test` target: `grep -s '^test:' Makefile`
    - Check for `go.mod`: use `go test ./...`
    - Check for `pytest.ini`, `setup.cfg`, or `pyproject.toml`: use `pytest`
    - Check for `Cargo.toml`: use `cargo test`
    - If none found: note in PR that no test suite was discovered

## Phase 4: Validate

14. Run the discovered test command.
15. If tests fail:
    - If the failure is in YOUR changes: fix and re-run (up to 2 times)
    - If the failure is pre-existing (unrelated to your changes): note it and proceed
16. Run `git diff` to review all changes and verify they match the feature description.

## Phase 5: Commit, Push & PR

17. Stage only the files you changed:
    ```
    git add <specific-files>
    ```
18. Commit with a descriptive message:
    ```
    git commit -m "Implement #<number>: <concise description>

    <brief explanation of what was implemented and approach taken>"
    ```
19. Push the branch:
    ```
    git push -u origin <current-branch>
    ```
20. Create a PR linking to the issue:
    ```
    gh pr create --title "Implement #<number>: <concise title>" --body "$(cat <<'EOF'
    ## Summary
    Implements #<number>

    **Feature**: <what was implemented>
    **Approach**: <brief description of approach taken>

    ## Changes
    - <bullet list of file changes>

    ## Test plan
    - [ ] Tests pass (or: No test suite discovered — manual testing required)
    - [ ] <specific test scenarios relevant to the feature>

    🤖 Generated with [Claude Code](https://claude.ai/claude-code)
    EOF
    )"
    ```

## Error Handling

- **Issue not found or inaccessible**: Stop and report the error. Do not proceed.
- **Cannot determine feature scope**: Stop. Add a comment to the issue explaining the ambiguity.
- **Scope too large (>10 files)**: Comment on issue and exit gracefully (see Phase 2 scope check).
- **Tests fail after implementation**: Fix up to 2 times. If still failing: stop, open PR with failing tests noted.
- **No test command discovered**: Non-blocking. Note in PR body that no test suite was found.
- **Merge conflicts**: `git fetch origin` and retry.

## Constraints

- This runs non-interactively. Do NOT ask for user input at any point.
- Keep scope minimal — implement only what the issue describes. No bonus features.
- Do not create new branches — work on the current worktree branch.
- Do not modify CI/CD, deployment configs, or unrelated files.
- Do not force-push or use destructive git operations.
- If the feature scope is unclear or ambiguous, comment on the issue and stop.
