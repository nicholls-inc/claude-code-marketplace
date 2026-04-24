# Acceptance-Oracle Scenario Schema

This document defines the YAML/JSON schema for scenarios emitted by `/acceptance-oracle-draft`, plus a runner-script stub and the mechanical-verification-only rule with worked rewrites.

## Schema (YAML, primary format)

```yaml
# acceptance/scenarios/<flow-name>.yaml
name: <kebab-case-identifier>         # required; unique across the scenario directory
surface: cli | http | daemon | github | ui   # required
priority: P0 | P1 | P2                # required; P0 failures block merge
timeout_seconds: <int>                # required; wall-clock budget
given:                                # preconditions (reproducible on a clean CI runner)
  env:                                # optional map of env vars to set
    KEY: value
  files:                              # optional list of fixture files to materialise
    - path: fixtures/input.json
      content_sha256: <hash>          # or inline `content: |` for small fixtures
  repo_state:                         # optional git-state preconditions
    clean: true
when:                                 # the action; exactly one of `cmd`, `http`, `signal`
  cmd:                                # shell invocation for CLI scenarios
    argv: ["mytool", "build", "--out", "dist"]
    stdin: ""                         # optional
  # OR
  http:                               # HTTP request for API scenarios
    method: POST
    url: http://localhost:8080/api/v1/jobs
    headers:
      Content-Type: application/json
    body_file: fixtures/input.json    # or `body: |` inline
  # OR
  signal:                             # daemon/background signal
    action: restart | health_check | emit_event
    target: <daemon-name>
then:                                 # list of mechanically-verifiable assertions
  - exit_code: 0                      # CLI only
  - stdout_matches: '^build complete in \d+ms$'   # regex, line-anchored by default
  - stderr_empty: true
  - stderr_matches: 'config file .+ not found'    # regex over stderr lines
  - http_status: 201                  # HTTP only
  - json_schema: schemas/job.schema.json           # path to a JSON Schema file
  - latency_under_ms: 300
  - file_exists: dist/bundle.js
  - file_sha256:
      path: dist/bundle.js
      hash: <expected-sha256>
  - log_line_within_seconds:
      pattern: 'daemon ready on port \d+'
      seconds: 5
```

### JSON equivalent

The same schema expressed as JSON, for teams that prefer it:

```json
{
  "name": "cli-build-happy-path",
  "surface": "cli",
  "priority": "P0",
  "timeout_seconds": 30,
  "given": {
    "env": {"MYTOOL_CONFIG": "fixtures/config.yaml"},
    "files": [{"path": "fixtures/input.json", "content_sha256": "abc123..."}]
  },
  "when": {
    "cmd": {"argv": ["mytool", "build", "--out", "dist"]}
  },
  "then": [
    {"exit_code": 0},
    {"stdout_matches": "^build complete in \\d+ms$"},
    {"file_exists": "dist/bundle.js"}
  ]
}
```

### Allowed assertion types (mechanical only)

| Assertion | Applies to | Signal |
|---|---|---|
| `exit_code: <int>` | cli | Process exit code matches. |
| `stdout_matches: <regex>` | cli | At least one stdout line matches. |
| `stdout_equals: <string>` | cli | Full stdout equals string. |
| `stderr_empty: true` | cli | stderr has zero bytes. |
| `stderr_matches: <regex>` | cli | At least one stderr line matches. |
| `http_status: <int>` | http | Response status equals. |
| `json_schema: <path>` | http, cli (JSON output) | Response/stdout parses and validates against the JSON Schema. |
| `header_matches` | http | Response header matches regex. |
| `latency_under_ms: <int>` | cli, http | Wall-clock under threshold. |
| `file_exists: <path>` | any | Path exists after `when`. |
| `file_sha256: {path, hash}` | any | File content SHA-256 matches. |
| `log_line_within_seconds: {pattern, seconds}` | daemon | A log line matching pattern appears within N seconds. |

If you need an assertion that isn't on this list, either add it here (and implement in the runner) or reject the scenario. **Do not** invent ad-hoc subjective assertions.

## Full example 1 — CLI scenario

```yaml
# acceptance/scenarios/cli-build-happy-path.yaml
name: cli-build-happy-path
surface: cli
priority: P0
timeout_seconds: 30
given:
  env:
    MYTOOL_CONFIG: fixtures/config.yaml
  files:
    - path: fixtures/config.yaml
      content: |
        target: es2022
        outdir: dist
when:
  cmd:
    argv: ["mytool", "build", "src/index.ts"]
then:
  - exit_code: 0
  - stdout_matches: '^build complete in \d+ms$'
  - file_exists: dist/index.js
  - latency_under_ms: 5000
```

## Full example 2 — HTTP API scenario

```yaml
# acceptance/scenarios/api-create-job.yaml
name: api-create-job
surface: http
priority: P0
timeout_seconds: 10
given:
  env:
    API_BASE_URL: http://localhost:8080
when:
  http:
    method: POST
    url: http://localhost:8080/api/v1/jobs
    headers:
      Content-Type: application/json
      Authorization: Bearer test-token
    body: |
      {"name": "smoke-job", "payload": {"x": 1}}
then:
  - http_status: 201
  - json_schema: schemas/job-created.schema.json
  - header_matches:
      name: Location
      pattern: '^/api/v1/jobs/[a-f0-9-]{36}$'
  - latency_under_ms: 500
```

## Runner-script stub (bash + jq, ~30 lines)

Emit this as `acceptance/run.sh` in the target repo. Pseudocode covers the common paths; extend per assertion.

```bash
#!/usr/bin/env bash
# GENERATED BY /acceptance-oracle-draft — extend per project needs.
set -uo pipefail   # intentionally NOT -e: scenario failures shouldn't abort the run.

SCENARIO_DIR="${SCENARIO_DIR:-acceptance/scenarios}"
REPORT="${REPORT:-acceptance/report.json}"
STRICT="${STRICT:-false}"   # true → P1/P2 failures also exit non-zero

pass=0 fail=0
results="[]"

for f in "$SCENARIO_DIR"/*.yaml; do
  name=$(yq '.name' "$f")
  prio=$(yq '.priority' "$f")
  timeout_s=$(yq '.timeout_seconds' "$f")
  surface=$(yq '.surface' "$f")
  verdict=skip; reason="unsupported surface: $surface"

  if [[ "$surface" == cli ]]; then
    # Read argv as a JSON array and exec it directly (no shell re-parsing).
    mapfile -t argv < <(yq -o=json '.when.cmd.argv[]' "$f" | jq -r '.')
    expected_rc=$(yq '.then[] | select(has("exit_code")) | .exit_code' "$f")
    timeout "$timeout_s" "${argv[@]}" >/tmp/oracle.out 2>&1
    rc=$?
    if [[ -n "$expected_rc" && "$rc" -ne "$expected_rc" ]]; then
      verdict=fail; reason="exit_code=$rc expected=$expected_rc"
    else
      verdict=pass; reason=""
      # TODO: evaluate stdout_matches, file_exists, latency_under_ms, etc.
    fi
  fi
  # TODO: implement surface=http (curl; http_status, json_schema, header_matches).
  # TODO: implement surface=daemon (log_line_within_seconds, health_check).

  case "$verdict" in
    pass) pass=$((pass + 1)) ;;
    fail) fail=$((fail + 1)) ;;
    skip) : ;;  # skipped scenarios don't count toward pass/fail
  esac
  results=$(jq --arg n "$name" --arg p "$prio" --arg v "$verdict" --arg r "$reason" \
    '. + [{name: $n, priority: $p, verdict: $v, reason: $r}]' <<<"$results")
  echo "[$verdict] $name ($prio) $reason"
done

jq --argjson p "$pass" --argjson f "$fail" '{summary: {pass: $p, fail: $f}, scenarios: .}' \
  <<<"$results" > "$REPORT"

# Fail the run if any P0 failed, or any failure in strict mode.
if jq -e '.[] | select(.verdict=="fail" and .priority=="P0")' <<<"$results" >/dev/null; then exit 1; fi
if [[ "$STRICT" == "true" && "$fail" -gt 0 ]]; then exit 1; fi
```

Dependencies: `yq` (mikefarah, YAML→JSON), `jq`, `curl` (for HTTP scenarios), `timeout`. Document these in the PR description.

## Mechanical-verification-only rule

Every scenario's `then` must compile to a deterministic predicate on observable process/file/network state. If you catch yourself writing a `then` that requires a human to judge it, stop and rewrite — or reject.

### Worked rewrites (subjective → mechanical)

**Example 1.**

- Rejected: *"CLI startup feels fast."*
- Rewritten: `latency_under_ms: 300` on `mytool --help`, measured via `hyperfine --warmup 3 mytool --help` on CI baseline hardware. Document the baseline hardware in the PR so threshold drift isn't misread as regression.

**Example 2.**

- Rejected: *"Help output is readable."*
- Rewritten: three mechanical checks — `stdout_matches: '^USAGE:'`, `stdout_matches: '^COMMANDS:'`, `stdout_matches: '^FLAGS:'`. Readability as a whole stays subjective, but section-presence is a sharp signal for the regression you actually care about (someone deleted a help section).

**Example 3.**

- Rejected: *"Error message is helpful when config is missing."*
- Rewritten: two mechanical checks — `exit_code: 2` (conventional for usage errors) and `stderr_matches: 'config file .+ not found \(set MYTOOL_CONFIG or pass --config\)'`. The regex enforces that the error names the missing path and tells the user how to fix it. "Helpful" as a vibe is discarded; "contains path + hint" is what actually matters.

### When to reject rather than rewrite

If none of the mechanical assertion types in the table above can express what you're trying to check, reject the flow. Candidates:

- Pure-visual correctness ("the chart looks right") — no programmatic observable.
- "Feels polished" — no measurable quantity.
- "Conforms to brand guidelines" — requires a human comparator.

Put rejected flows in the `## Rejected Flows` section of the skill output (see `SKILL.md` Step 7) so the coverage boundary is explicit. Do not quietly drop them; the user needs to know what the oracle isn't catching.
