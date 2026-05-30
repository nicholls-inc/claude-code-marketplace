# Crosscheck conformance / inventory oracle

The doc-vs-implementation gate for Crosscheck *as a whole*. Same idea as the
invariant↔test coverage gate, lifted to the meta level: **docs ↔ artifacts.**
It exists because Crosscheck's own `plausible ≠ correct` failure mode bit its
maintainer — the docs described a methodology that only partly ships, and
nothing mechanical said which parts. This is that mechanical check.

It is a self-contained Go module (stdlib only, no third-party deps).

## Run

    go run ./crosscheck/conformance              # from repo root, scans crosscheck/
    go run ./crosscheck/conformance crosscheck   # explicit root
    go run ./crosscheck/conformance /path/to/crosscheck

Build a standalone binary:

    go build -o conformance ./crosscheck/conformance
    ./conformance crosscheck

Exit 0 = PASS, 1 = FAIL (any AUTO error, or any `unreviewed` ledger claim, or a
`present_artifact` ledger check that disagrees with the filesystem).

> Run commands assume the repo-root Go workspace (`go.work`), which lets the
> nested module resolve when invoked from the repo root. From inside this
> directory you can also run `go run . ..` and `go test ./...`.

## Two layers

- **AUTO (deterministic, fails CI):**
  1. *Structural* — every `skills/<dir>/` has a `SKILL.md` with `name`/`description`
     frontmatter; `name` matches the dir; agents likewise. Non-empty.
  2. *Phantom* — docs reference a `/skill` or `/crosscheck:skill` that doesn't exist.
  3. *Orphan* — an artifact ships but is referenced in no user-facing doc (WARN).
  4. *MCP* — README-claimed `dafny_*` tools exist in `mcp-server/` source (WARN).
- **LEDGER (`claims.json`, reviewed not auto-proved):** narrative claims that
  can't be checked by reference integrity — layer/phase/mode counts, terminal
  states, self-coverage. Each entry records the claim, the observed reality, a
  review `status`, and where possible an auto-check that **re-fires if reality
  changes** (e.g. `agents/lowry.md expect_present:false` flips to FAIL the day
  Phase 4 ships, forcing the ledger to be updated). `status:"unreviewed"` fails
  CI to force triage of any new claim. A `status:"known-gap"` entry must carry a
  `tracked_in` link to its section in the ledger-gap roadmap
  ([`../docs/add/roadmap.md`](../docs/add/roadmap.md)); a known-gap with no link
  also fails CI, so a "known" gap can never be tracked nowhere.

## First-run findings (2026-05-30, plugin v2.5.1)

- **ERROR** `assurance-probe` — `SKILL.md` had no frontmatter; couldn't load as a
  skill, yet `byfuglien` routes to it. **Fixed in the PR that introduced this
  oracle** (frontmatter added; this is why the gate is now GREEN).
- **WARN** `journal-context` — was undocumented in the user-facing doc set; now
  documented in the README skills overview and the top-level `CLAUDE.md`, so the
  orphan warning clears.
- **LEDGER known-gaps** — Phase 4 agent, operating modes, committed methodology,
  Phase 5 auditor, self-coverage. See `claims.json`; each is tracked in the
  ledger-gap roadmap ([`../docs/add/roadmap.md`](../docs/add/roadmap.md)).

## Wire to CI (suggested)

```yaml
# .github/workflows/conformance.yml  (sketch)
jobs:
  conformance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go run ./crosscheck/conformance crosscheck
      - run: go test ./crosscheck/conformance/...
```

## Extend

Add a claim to `claims.json` whenever the docs assert something about the plugin
that the filesystem doesn't already prove. New claims start `status:"unreviewed"`
(fails CI) until a human triages them to `reviewed-*` or `known-gap`.
