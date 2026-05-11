# claude-github-app

A `claude` wrapper that injects a short-lived GitHub App installation token chosen by working directory, so PRs opened from inside a Claude Code session are authored by the App bot (`my-app[bot]`). You can then approve those PRs from your personal GitHub account because GitHub's self-approval rule applies to the PR author, not the human running the wrapper.

## Install

```bash
cd tools/claude-github-app
make install          # builds bin/claude + bin/claude-github-app and copies to ~/bin/
```

Add `~/bin` to PATH ahead of `~/.local/bin` (where the real `claude` lives):

```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
exec zsh
```

Verify shadowing:

```bash
which -a claude
# /Users/you/bin/claude
# /Users/you/.local/bin/claude     # real binary; shadowed
```

### Optional: install gh + git shims

Installation tokens last ~1h. Once `claude` is exec'd, its inherited `GH_TOKEN` is frozen in the child process — a 90-minute coding session will hit `gh` 401s and `git push` auth failures at the 1h mark. The shims fix this:

```bash
make install-shims    # builds bin/gh + bin/git and copies to ~/bin/
# (or `make install-all` for everything in one go)
```

`~/bin/gh` and `~/bin/git` shadow the real binaries on PATH. Each invocation re-mints from `~/.cache/claude-github-app/<app>.json` if the token is within the 5-minute refresh window. Verify:

```bash
which -a gh git
# /Users/you/bin/gh         (shim)
# /opt/homebrew/bin/gh      (real, found by walking PATH and skipping ~/bin)
# /Users/you/bin/git        (shim)
# /usr/bin/git              (real)
```

**The shims pass through unmodified when CWD doesn't match any `[[mappings]]` entry.** Running `gh repo view` from a non-mapped directory is functionally identical to running real `gh` directly — no token injection, no env changes, no extra GitHub API calls.

Inside a mapped directory, the shims:
- **gh shim**: overrides `GH_TOKEN` and `GITHUB_TOKEN` in the env passed to real `gh`.
- **git shim**: writes a fresh gitconfig at `~/.cache/claude-github-app/<app>-gitconfig` and sets `GIT_CONFIG_GLOBAL` to it before exec'ing real `git`.

Cost per invocation: ~5ms on cache hit, ~500ms when minting fresh (≤ once per hour per app).

## One-time GitHub App setup

1. Create a GitHub App (Settings → Developer settings → GitHub Apps → New).
2. Permissions you'll likely want (set on the App, then accepted by each repo at install time):
   - `contents: write` — push branches
   - `pull_requests: write` — open/edit PRs
   - `issues: write` — comment, label
   - `workflows: write` — only if Claude will edit `.github/workflows/*.yml`
3. Generate a private key, save as `~/.config/claude-github-app/keys/<app-slug>.pem`, then `chmod 600` it. The wrapper refuses to launch if the key is group/other-readable.
4. Install the App on the repos you want to use it with. Record the **installation ID** from the App's installations page (or `GET /app/installations` with an App JWT).
5. Optionally, configure branch protection: require at least one PR review. Make sure the App itself does NOT have "bypass branch protection" rights.

## Configuration

`~/.config/claude-github-app/config.toml`:

```toml
# Optional. If unset, the wrapper resolves the real claude binary by reading
# ~/.local/bin/claude as a symlink (the macOS install layout).
# claude_binary = "/Users/you/.local/bin/claude"

# Optional. App to use when the CWD doesn't match any [[mappings]] entry.
# Omit/empty for "no github app auth'd" in unmapped directories.
# default_app = "personal"

[[apps]]
name             = "my-app"            # MUST match the GitHub App slug exactly
client_id        = "Iv23li..."         # Client ID or App ID — GitHub accepts either as JWT iss
installation_id  = 12345678
private_key_file = "~/.config/claude-github-app/keys/my-app.pem"

# Optional: scope tokens to specific repos by GitHub repository ID.
# repository_ids = [111111, 222222]

# Optional: skip the GET /users/<slug>[bot] lookup by hard-coding the bot ID.
# bot_user_id = 198765432

# Optional: explicit permissions map. Default:
#   contents:write, pull_requests:write, issues:write, workflows:write
# [apps.permissions]
# contents      = "write"
# pull_requests = "write"

[[mappings]]
path = "/Users/you/repos/formal-verify"
app  = "my-app"

# Git worktrees do NOT inherit mappings. Add separate entries for each
# worktree path you launch claude from:
# [[mappings]]
# path = "/Users/you/repos/formal-verify-feature-x"
# app  = "my-app"
```

Mappings are matched by **longest directory prefix** of the current working directory; on miss, the wrapper retries with `filepath.EvalSymlinks(cwd)` (so symlinked working trees work). Trailing slashes are normalised. Duplicate `path` entries are rejected.

## Usage

Just run `claude` from a configured directory:

```bash
cd ~/repos/formal-verify
claude
# claude-github-app: using github app 'my-app' for PR + commit identity (installation 12345678, expires 15:14:00 UTC)
```

Inside the session, verify:

```bash
gh auth status                 # Logged in to github.com via GH_TOKEN
gh api user                    # { "login": "my-app[bot]", ... }
git config user.email          # 198765432+my-app[bot]@users.noreply.github.com
```

`gh pr create` will now author the PR as `my-app[bot]`.

### Mid-session token refresh

Installation tokens last ~1 hour. The wrapper mints fresh at launch only — once `claude` is running, its inherited `GH_TOKEN` cannot be replaced by the parent.

**Recommended: install the gh + git shims** (see "Optional: install gh + git shims" above). With shims installed, every `gh` and `git` invocation auto-refreshes from cache — no user action needed, no restart required.

Without the shims, the manual workarounds below still apply:

```bash
# One-shot fresh token for a single gh command
GH_TOKEN=$(claude-github-app token) gh pr create ...

# Rewrite the live $GIT_CONFIG_GLOBAL so git push picks up a new bearer
claude-github-app git-config-refresh
```

For a persistently fresh `GH_TOKEN` inside the claude process itself, restart `claude`. The wrapper will re-mint.

### Inspecting cached state

```bash
claude-github-app status
# my-app                          fresh   expires 2026-05-11T16:14:00Z (54m23s left)

tail -5 ~/.cache/claude-github-app/status.log
# 2026-05-11T15:14:00Z using github app 'my-app' ...
```

## Contract

The wrapper guarantees the following invariants. See `docs/plan.md` in this PR for the full design.

**§A — Token confidentiality.** Token-bearing files are mode 0600 in directories of mode 0700. The token is never written to stdout, argv, or any log line. (Residual: env vars are visible via `ps -E` — unavoidable for `gh`-compatible flows.)

**§B — Identity post-conditions.**
- **PR-actor:** when an app is mapped and token mint succeeds, `gh pr create` produces a PR with `author.login == "<app>[bot]"`.
- **Commit-author:** when the bot user ID is known, commits have `author.email = "<id>+<app>[bot]@users.noreply.github.com"`. If bot lookup fails, the wrapper degrades commit-author silently but flags it in the status line.

**§C — Token lifetime.** The wrapper guarantees a valid token only at the moment of `exec`. Tokens expire ~1h later per GitHub policy. The optional gh/git shims relax this to "valid at every `gh`/`git` invocation" by re-minting from the shared cache. For tools other than `gh`/`git`, restart `claude` for a persistently fresh token in-process or use `claude-github-app token` per call.

**§D — Cleanup.** Temp dirs are removed on normal exit, signal-forwarded exit, or recovered panic. Under `SIGKILL` / host crash, residency is bounded by the 24h startup sweep.

**§E — Negative space.** The wrapper never modifies `~/.config/gh/`, `~/.gitconfig`, or `~/.netrc`. It never invokes `gh auth login`. It never writes the token to any file outside its 0600 cache.

## Threat model

The wrapper trusts: the invoking user; the contents of `~/.config/claude-github-app/`; the resolved real-claude binary; the user's HOME with normal ACLs.

The wrapper does NOT defend against:
- Malicious code executed by `claude` itself (deferred to Claude Code permission rules).
- Tampering with `~/bin/claude` or `~/.local/bin/claude` by an attacker with write access.
- Invocation under `sudo` — refused with a clear error.
- Prompt injection extracting `GH_TOKEN` via tool calls. Mitigate by scoping `repository_ids` and minimising the `permissions` map.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `private key file ... is mode 0644` | PEM perms too broad | `chmod 600 ~/.config/claude-github-app/keys/<app>.pem` |
| `bot user lookup failed for X` | App slug ≠ `name` in config | Make sure `name` matches the GitHub App's slug (lowercase, hyphenated) exactly |
| `gh` 401 after ~1h | Token expired mid-session | Install the shim (`make install-shims`), or `GH_TOKEN=$(claude-github-app token) gh ...` |
| Shim runs but token isn't injected | CWD doesn't match any `[[mappings]]` | Add a mapping for the CWD, or set `default_app` in config |
| `which gh` returns real gh first | `~/bin` not on PATH before `/opt/homebrew/bin` etc. | Put `export PATH="$HOME/bin:$PATH"` early in `~/.zshrc` |
| `which claude` returns `/Users/.../.local/bin/claude` first | `~/bin` not on PATH (or not first) | Add `export PATH="$HOME/bin:$PATH"` to the start of `~/.zshrc` |
| PR author is your personal account | `gh` used cached creds, not `GH_TOKEN` | Confirm `gh auth status` shows `via GH_TOKEN`. Check no `GITHUB_TOKEN` was unset by an outer wrapper |
| Wrapper hangs after Ctrl-Z | Job control bug | `fg` to resume both; report as bug |

## Development

```bash
make test         # go test ./... -race -count=1
make vet
make tidy
make build        # produces bin/{claude,claude-github-app}
make clean
```

No Docker required.

## Out of scope (v1)

- Claude Code permission deny rules / PreToolUse hooks (defense in depth).
- macOS Keychain storage for the App private key.
- Multi-platform builds (v1 is macOS/zsh; Linux likely works but is untested).
- Auto-detection of git worktrees back to their main repo path.
