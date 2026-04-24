# Go Template — Invariant Coverage Gate

Emit a Go tool that parses invariant IDs from `docs/invariants/**/*.md` and cross-checks them against `// Invariant <ID>: <Name>` comments in the repo's test files. Placeholders are marked `<placeholder>`.

## Script Template

Write to `scripts/check_invariant_coverage.go` (runnable via `go run`). For repos with a `cli/` tree, prefer `cli/cmd/check-invariants/main.go`.

```go
//go:build ignore

// Bidirectional invariant-to-test coverage gate.
// Run: go run scripts/check_invariant_coverage.go
// Exits 0 on success, 1 on any coverage gap or orphan comment.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	headerRe       = regexp.MustCompile(`^\*\*([A-Z]+\d+[a-z]?)\.\s`)    // "**I1. Name.**"
	commentRe      = regexp.MustCompile(`^//\s*Invariant\s+([A-Z]+\d+[a-z]?):\s`) // "// Invariant I1: Name."
	aspirationalRe = regexp.MustCompile(`<!--\s*aspirational\s*-->`)

	testSuffix = "<test-suffix>" // e.g. "_invariants_prop_test.go"
	invDir     = "docs/invariants"
)

type entry struct {
	module, id, path string
	line             int
	aspirational     bool
}

func scan(path string, re *regexp.Regexp) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []entry
	s := bufio.NewScanner(f)
	for ln := 1; s.Scan(); ln++ {
		if m := re.FindStringSubmatch(s.Text()); m != nil {
			out = append(out, entry{
				id: m[1], path: path, line: ln,
				aspirational: aspirationalRe.MatchString(s.Text()),
			})
		}
	}
	return out, s.Err()
}

func main() {
	var invariants, comments []entry
	for _, f := range must(filepath.Glob(filepath.Join(invDir, "*.md"))) {
		if b := filepath.Base(f); b == "README.md" || b == "COVERAGE.md" {
			continue
		}
		es := must(scan(f, headerRe))
		for i := range es {
			es[i].module = strings.TrimSuffix(filepath.Base(f), ".md")
		}
		invariants = append(invariants, es...)
	}
	_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(p, testSuffix) {
			return nil
		}
		es := must(scan(p, commentRe))
		for i := range es {
			es[i].module = filepath.Base(filepath.Dir(p))
		}
		comments = append(comments, es...)
		return nil
	})

	covered := map[string]bool{}
	for _, c := range comments {
		covered[c.module+"/"+c.id] = true
	}
	declared := map[string]bool{}
	for _, i := range invariants {
		declared[i.module+"/"+i.id] = true
	}

	var missing, orphans []entry
	for _, i := range invariants {
		if !i.aspirational && !covered[i.module+"/"+i.id] {
			missing = append(missing, i)
		}
	}
	for _, c := range comments {
		if !declared[c.module+"/"+c.id] {
			orphans = append(orphans, c)
		}
	}

	if len(missing) == 0 && len(orphans) == 0 {
		fmt.Println("invariant coverage OK")
		return
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "Missing coverage — add `// Invariant <ID>: <Name>` above the property test:")
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "  - %s/%s declared at %s:%d (no covering test)\n", m.module, m.id, m.path, m.line)
		}
	}
	if len(orphans) > 0 {
		fmt.Fprintln(os.Stderr, "Orphan test comments — ID not declared in any invariant doc:")
		for _, o := range orphans {
			fmt.Fprintf(os.Stderr, "  - %s/%s at %s:%d\n", o.module, o.id, o.path, o.line)
		}
	}
	fmt.Fprintln(os.Stderr, "\nFix: add the missing comment, remove the orphan, or mark the invariant `<!-- aspirational -->` in its doc header.")
	os.Exit(1)
}

func must[T any](v T, err error) T {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	return v
}
```

Replace `<test-suffix>` with the detected suffix (e.g. `_invariants_prop_test.go`). Dependency-free — standard library only. Requires Go 1.18+ for generics.

## Pre-commit

**pre-commit.com (`.pre-commit-config.yaml`):**

```yaml
- repo: local
  hooks:
    - id: invariant-coverage
      name: Invariant coverage (Go)
      entry: go run scripts/check_invariant_coverage.go
      language: system
      pass_filenames: false
      files: '^(docs/invariants/.*\.md|.*_invariants_prop_test\.go)$'
```

**lefthook (`lefthook.yml`):** add under `pre-commit.commands.invariant-coverage` with `glob: 'docs/invariants/*.md,**/*_invariants_prop_test.go'` and `run: go run scripts/check_invariant_coverage.go`.

**husky (`.husky/pre-commit`):** append `go run scripts/check_invariant_coverage.go || exit 1`.

**Standalone (`scripts/pre-commit-invariant-coverage.sh`):** `#!/usr/bin/env bash`, `set -euo pipefail`, then the `go run` line. Wire via `git config core.hooksPath .githooks` plus a symlink.

## CI

**GitHub Actions (`.github/workflows/invariant-coverage.yml`):**

```yaml
name: invariant-coverage
on: [pull_request, push]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go run scripts/check_invariant_coverage.go
```

**GitLab CI (`.gitlab-ci.yml` job):**

```yaml
invariant-coverage:
  image: golang:1.22
  script: [go run scripts/check_invariant_coverage.go]
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH
```

## Error Message

On failure the script prints to stderr:

```
Missing coverage — add `// Invariant <ID>: <Name>` above the property test:
  - queue/I1 declared at docs/invariants/queue.md:42 (no covering test)

Orphan test comments — ID not declared in any invariant doc:
  - runner/IX at cli/internal/runner/runner_invariants_prop_test.go:88

Fix: add the missing comment, remove the orphan, or mark the invariant `<!-- aspirational -->` in its doc header.
```

The fix command is embedded so an AI coding agent reading the failure can act without human translation.
