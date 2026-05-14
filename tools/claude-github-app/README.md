# claude-github-app

**Let Claude Code open PRs as a GitHub App bot so you can approve them from your own account.**

When you run `claude` inside a configured repo, this wrapper swaps in a short-lived GitHub App installation token. PRs opened during the session are authored by `your-app[bot]` — not you. Branch protection's "no self-approval" rule applies to the PR author, so reviewing and approving your agent's work from your own GitHub account is no longer a policy violation; it's just code review.

That's the whole pitch. The rest of this README is the install, the contract, and the honest list of things it doesn't do.

## Is this for you?

You'll get value from this if you:

- Run Claude Code (or any agent) against repos with branch protection that blocks self-approval.
- Want a clean audit trail where bot-authored commits and PRs look like bot-authored commits and PRs.
- Are on macOS with zsh. (Linux probably works. Nobody has tested it. PRs welcome.)

You probably don't need this if you:

- Don't use branch protection, or your protection rules don't require review.
- Already drive Claude Code through GitHub Actions or another CI-side flow — that runs as the App bot natively.
- Want a hardened, multi-user, secrets-management-grade tool. This is a single-user dev convenience.

## What it isn't

- **Not a Claude Code plugin.** It's a binary that sits in front of `claude` on your PATH.
- **Not a credential manager.** It doesn't touch `~/.config/gh/`, `~/.gitconfig`, or `~/.netrc`. It doesn't run `gh auth login`. It writes one token file to its own cache, mode 0600, and gets out of the way.
- **Not a sandbox.** If the agent inside `claude` decides to do something destructive with the token, the wrapper won't stop it. Scope the App's permissions and `repository_ids` to limit blast radius.
- **Not cross-platform yet.** v1 is macOS/zsh. Linux probably works, but no one has confirmed.

## How it works (90 seconds)

1. `~/bin/claude` shadows the real `claude` binary on your `PATH`.
2. On launch, it reads your CWD, looks up the longest matching `[[mappings]]` entry in `~/.config/claude-github-app/config.toml`, and mints an installation token for that GitHub App.
3. It exports `GH_TOKEN`, writes a scoped gitconfig, then `exec`s the real `claude` with a clean isolated environment (`GH_CONFIG_DIR`, `GIT_CONFIG_GLOBAL`).
4. Inside the session, `gh` and `git` see the bot's identity. PRs are authored by `your-app[bot]`.
5. **Optional shims** for `gh` and `git` re-mint from a shared cache on every invocation, so tokens never go stale during long sessions.

No daemon. No background process. One token cache, mode 0600, under `~/.cache/claude-github-app/`.

## Install

```bash
cd tools/claude-github-app
make install          # wrapper only
# or
make install-all      # wrapper + gh/git shims (recommended)
```

This installs to `~/bin/`. Put `~/bin` first on your `PATH`:

```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
exec zsh
```

Confirm the shadow:

```bash
$ which -a claude
/Users/you/bin/claude              # wrapper (this tool)
/Users/you/.local/bin/claude       # real binary, now shadowed
```

If the real `claude` shows up first, the wrapper won't run. Reorder your `PATH` and try again.

## Optional but recommended: `gh` + `git` shims

GitHub installation tokens last about an hour. The wrapper only mints once, at launch — so a 90-minute coding session reliably ends with `gh: 401 Bad credentials` and `git push` auth failures around the one-hour mark. The shims fix that:

```bash
make install-shims    # adds ~/agent-bin/gh and ~/agent-bin/git
# (already included if you ran `make install-all`)
```

The shims install to `~/agent-bin/` rather than `~/bin/` so they don't shadow the real `gh`/`git` in your interactive shell. They only intercept inside contexts that prepend `~/agent-bin` to `PATH`. For Claude Code, add it to `env.PATH` in `~/.claude/settings.json`:

```json
{
  "env": {
    "PATH": "/Users/you/agent-bin:/usr/local/bin:/usr/bin:/bin"
  }
}
```

Every `gh` or `git` invocation re-reads the shared cache at `~/.cache/claude-github-app/<app>.json` and re-mints if the token is within five minutes of expiry. Cost: ~5ms on a cache hit, ~500ms when minting fresh (at most once per app per hour).

The shims are deliberately boring. **If your current directory isn't mapped to an app, the shim passes through to the real `gh` or `git` with no modifications.** Running `gh repo view` from an unmapped directory behaves identically to running real `gh` directly. No token injection, no env changes, no extra API calls.

Inside a mapped directory, the shims do the minimum needed:

- `gh` shim: sets `GH_TOKEN` and `GITHUB_TOKEN` in the env passed to real `gh`.
- `git` shim: writes a fresh gitconfig to the cache and points `GIT_CONFIG_GLOBAL` at it before exec'ing real `git`.

Verify after install, from a context where `~/agent-bin` is on `PATH` (e.g. inside a Claude Code session with the `env.PATH` above):

```bash
$ which -a gh git
/Users/you/agent-bin/gh   # shim
/opt/homebrew/bin/gh      # real, found by walking PATH past ~/agent-bin
/Users/you/agent-bin/git  # shim
/usr/bin/git              # real
```

In your regular interactive shell, `which gh` should resolve straight to the real binary — that's intentional.

## One-time GitHub App setup

Do this once per App.

1. **Create the App.** Settings → Developer settings → GitHub Apps → New GitHub App.
2. **Set permissions** on the App (each repo accepts them at install time):
   - `contents: write` — push branches
   - `pull_requests: write` — open and edit PRs
   - `issues: write` — comment, label
   - `workflows: write` — only if Claude will edit files under `.github/workflows/`
3. **Generate and save the private key.**
   ```bash
   mkdir -p ~/.config/claude-github-app/keys
   mv ~/Downloads/my-app.*.private-key.pem ~/.config/claude-github-app/keys/my-app.pem
   chmod 600 ~/.config/claude-github-app/keys/my-app.pem
   ```
   The wrapper refuses to launch if the key is group- or world-readable.
4. **Install the App on the repos you want.** From the App's settings page, click Install. Note the **installation ID** (visible in the URL after install, or via `GET /app/installations` with an App JWT).
5. **Configure branch protection** to require at least one PR review. Confirm the App itself does **not** have "bypass branch protection".

## Configuration

Everything lives in `~/.config/claude-github-app/config.toml`:

```toml
# Optional. If unset, the wrapper resolves the real claude binary by reading
# ~/.local/bin/claude as a symlink (the macOS install layout).
# claude_binary = "/Users/you/.local/bin/claude"

# Optional. App to use when the CWD doesn't match any [[mappings]] entry.
# Omit for "no GitHub App auth" in unmapped directories.
# default_app = "personal"

[[apps]]
name             = "my-app"            # MUST match the GitHub App slug exactly
client_id        = "Iv23li..."         # Client ID or App ID — both work as JWT iss
installation_id  = 12345678
private_key_file = "~/.config/claude-github-app/keys/my-app.pem"

# Optional: scope tokens to specific repo IDs (recommended for blast radius).
# repository_ids = [111111, 222222]

# Optional: skip the GET /users/<slug>[bot] lookup by hard-coding the bot ID.
# bot_user_id = 198765432

# Optional: override the default permissions map
# (contents/pull_requests/issues/workflows all "write").
# [apps.permissions]
# contents      = "write"
# pull_requests = "write"

[[mappings]]
path = "/Users/you/repos/formal-verify"
app  = "my-app"

# Git worktrees do NOT inherit mappings from their parent repo.
# Add a separate entry per worktree path.
# [[mappings]]
# path = "/Users/you/repos/formal-verify-feature-x"
# app  = "my-app"
```

The wrapper matches the CWD against `[[mappings]]` by **longest directory prefix**. On miss, it retries with `filepath.EvalSymlinks(cwd)` so symlinked working trees still match. Trailing slashes are normalised. Duplicate `path` entries are rejected at startup.

## Day-to-day use

Run `claude` from a configured directory. The startup line tells you which app got picked:

```bash
$ cd ~/repos/formal-verify
$ claude
claude-github-app: using github app 'my-app' for PR + commit identity
                   (installation 12345678, expires 15:14:00 UTC)
```

Sanity-check from inside the session:

```bash
$ gh auth status
github.com
  ✓ Logged in to github.com as my-app[bot] (GH_TOKEN)

$ gh api user | jq .login
"my-app[bot]"

$ git config user.email
198765432+my-app[bot]@users.noreply.github.com
```

`gh pr create` will now produce a PR authored by `my-app[bot]`.

### Mid-session token refresh

If you installed the shims, you don't need to do anything — skip this section.

If you didn't, the wrapper mints exactly once at launch and `claude` inherits `GH_TOKEN` at exec time — the parent can't reach back in to update it. Workarounds:

```bash
# One-shot fresh token for a single gh command
GH_TOKEN=$(claude-github-app token) gh pr create ...

# Rewrite the live $GIT_CONFIG_GLOBAL so the next git push picks up a new bearer
claude-github-app git-config-refresh
```

For a persistently fresh `GH_TOKEN` inside the process, restart `claude`.

### Inspecting cached state

```bash
$ claude-github-app status
my-app                          fresh   expires 2026-05-11T16:14:00Z (54m23s left)

$ tail -5 ~/.cache/claude-github-app/status.log
2026-05-11T15:14:00Z using github app 'my-app' ...
```

---

## What the wrapper guarantees

The contract below is what the wrapper enforces. The threat model after it is what's deliberately *not* in scope.

### Contract

**§A — Token confidentiality.** Token-bearing files are mode 0600 inside directories of mode 0700. Tokens never appear in stdout, argv, or log lines. *Residual exposure:* env vars are visible to other processes of the same user via `ps -E`. That's unavoidable for `gh`-compatible flows — tools read `GH_TOKEN` from the environment by design.

**§B — Identity post-conditions.**
- **PR-actor:** when an app is mapped and the token mint succeeds, `gh pr create` produces a PR with `author.login == "<app>[bot]"`.
- **Commit-author:** when the bot user ID is known, commits carry `author.email = "<id>+<app>[bot]@users.noreply.github.com"`. If the bot lookup fails (network blip, slug mismatch), the wrapper still launches but flags the degradation in the status line.

**§C — Token lifetime.** The wrapper guarantees a valid token *at the moment of `exec`*. GitHub expires tokens ~1h later. The optional shims upgrade this to "valid at every `gh`/`git` invocation" by re-minting from the shared cache. For other tools, restart `claude` or call `claude-github-app token` per invocation.

**§D — Cleanup.** Temp directories are removed on normal exit, signal-forwarded exit, or recovered panic. Under `SIGKILL` or a host crash, leftover state is bounded by the 24h startup sweep.

**§E — Negative space.** The wrapper never modifies `~/.config/gh/`, `~/.gitconfig`, or `~/.netrc`. It never invokes `gh auth login`. It never writes a token to any file outside its 0600 cache.

### Threat model

The wrapper trusts: the invoking user, the contents of `~/.config/claude-github-app/`, the resolved real-claude binary, and the user's HOME with normal Unix permissions.

The wrapper does **not** defend against:

- **Malicious code executed by `claude` itself.** Once the agent has the token, any tool call can use it. Mitigate with Claude Code's permission rules and by scoping `repository_ids` + `permissions` tightly.
- **Prompt injection that extracts `GH_TOKEN`** via tool output or exfil. Same mitigation: narrow `repository_ids` and `permissions`.
- **Tampering with `~/bin/claude`** by an attacker who already has write access to your home directory.
- **Invocation under `sudo`.** Refused with a clear error rather than guessing at someone else's HOME.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `private key file ... is mode 0644` | PEM permissions too loose | `chmod 600 ~/.config/claude-github-app/keys/<app>.pem` |
| `bot user lookup failed for X` | App slug doesn't match `name` in config | Make `name` match the GitHub App slug exactly (lowercase, hyphenated) |
| `gh` returns 401 after ~1h | Installation token expired mid-session | Install shims (`make install-shims`) and ensure `~/agent-bin` is on `PATH` in that context, or `GH_TOKEN=$(claude-github-app token) gh ...` |
| Shim runs but doesn't inject a token | CWD has no matching `[[mappings]]` entry | Add a mapping, or set `default_app` in config |
| `which gh` returns real `gh` first | `~/bin` not first on PATH | Put `export PATH="$HOME/bin:$PATH"` early in `~/.zshrc` |
| `which claude` shows `~/.local/bin/claude` first | Same — PATH ordering | Same fix |
| PR is authored by your personal account | `gh` used cached creds, not `GH_TOKEN` | Confirm `gh auth status` says `via GH_TOKEN`; check nothing unset `GITHUB_TOKEN` |
| Wrapper hangs after Ctrl-Z | Job-control bug | `fg` to resume; report as a bug |

## Development

Pure Go. No Docker. No external services needed for the test suite.

```bash
make test         # go test ./... -race -count=1
make vet
make tidy
make build        # produces bin/{claude,claude-github-app}
make clean
```

## Out of scope (v1)

Things I considered and consciously left out:

- **Claude Code permission deny rules / PreToolUse hooks** for defense in depth. Use Claude Code's own mechanisms.
- **macOS Keychain** storage for the App private key. Files + perms, for now.
- **Multi-platform builds.** macOS/zsh only. Linux probably works; nobody has tested it.
- **Auto-detection of git worktrees** back to their main repo path. Add a `[[mappings]]` entry per worktree.

For the full design rationale and contract derivation, see `docs/plan.md`.
