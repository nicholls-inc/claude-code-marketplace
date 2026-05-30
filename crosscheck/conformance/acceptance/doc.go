// Package acceptance holds the A1–A6 acceptance oracles: the behavioral
// contracts the (not-yet-built) ADD Phase 4 implementation loop must run to
// green. They are DRAFT and PENDING RATIFICATION — see RATIFY.md.
//
// # Why these are not in the blocking CI lane
//
// Every check file in this package carries the build constraint
//
//	//go:build acceptance
//
// so the default build the conformance job runs — `go vet ./...`,
// `go test ./...`, `go run . ..` from crosscheck/conformance — never compiles
// or executes them. Without that, the acceptance package (whose checks fail by
// design) would turn the blocking conformance gate red. This file (doc.go) and
// oracles.go are intentionally UNtagged so the package still has buildable Go
// files under the default tags; otherwise `go vet ./...` would error with
// "build constraints exclude all Go files".
//
// # Running the acceptance lane
//
//	go test -tags acceptance ./acceptance/...   # from crosscheck/conformance
//
// All A1–A6 checks currently FAIL or report PENDING by design. They turn green
// only once (a) the maintainer ratifies the contracts in RATIFY.md and (b) the
// Phase 4 build + the judged-oracle harness exist. Per ADR-002 the oracles
// split two ways: deterministic (A3, A4 — pure functions over repo state /
// commit grammar) and judged (A1, A2, A5, A6 — scripted scenario runs scored
// by an LLM judge, seeded from real field reports so they are not synthetic).
//
// THIS PACKAGE CHANGES NO ORCHESTRATOR, AGENT, OR SKILL BEHAVIOR. It does not
// build the Phase 4 agent, the mode system, the commit-shape classifier, or the
// judged-oracle runner. It only states, as runnable red checks, what those must
// satisfy.
package acceptance
