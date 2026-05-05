---

## gh Auth Verification Report

---

### Step 1 — `which gh`

**Command:** `which gh`
**Exit code:** `0`

```
/usr/local/bin/gh
```

---

### Step 2 — `gh --version`

**Command:** `gh --version`
**Exit code:** `1`

```
stderr: ERROR: GITHUB_APP_PRIVATE_KEY is not set or empty — required to mint GitHub App tokens
```

> The custom wrapper at `/usr/local/bin/gh` intercepts **all** `gh` invocations (including `--version`) and immediately errors if `GITHUB_APP_PRIVATE_KEY` is not set. This means the image ships a credential-minting shim rather than the raw `gh` binary, and that shim requires a GitHub App private key to be injected at runtime.

---

### Step 3 — `gh auth status`

**Command:** `gh auth status`
**Exit code:** `1`

```
stderr: ERROR: GITHUB_APP_PRIVATE_KEY is not set or empty — required to mint GitHub App tokens
```

---

### Step 4 — `gh api user --jq .login`

**Command:** `gh api user --jq .login`
**Exit code:** `1`

```
stderr: ERROR: GITHUB_APP_PRIVATE_KEY is not set or empty — required to mint GitHub App tokens
```

---

### Step 5 — `gh repo view nicholls-inc/fabro-agent-base --json nameWithOwner,visibility`

**Command:** `gh repo view nicholls-inc/fabro-agent-base --json nameWithOwner,visibility`
**Exit code:** `1`

```
stderr: ERROR: GITHUB_APP_PRIVATE_KEY is not set or empty — required to mint GitHub App tokens
```

---

VERDICT: gh auth FAILED — `GITHUB_APP_PRIVATE_KEY` env var is not set; the image's `gh` wrapper cannot mint a GitHub App token and every command fails before reaching the GitHub API.