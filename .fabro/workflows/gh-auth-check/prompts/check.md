# gh auth check

You are running inside a Fabro sandbox built from
`ghcr.io/nicholls-inc/fabro-agent-base`. The only goal of this run is to
confirm that the `gh` CLI is installed and authenticated. Do not modify
any files. Do not call any other tools.

## Steps

1. Run `which gh` and report the path (or that it is missing).
2. Run `gh --version` and report the output.
3. Run `gh auth status` and report the full output verbatim.
4. Run `gh api user --jq .login` and report the returned login
   (or the error if it fails).
5. Run `gh repo view nicholls-inc/fabro-agent-base --json nameWithOwner,visibility`
   and report the output (this exercises an authenticated read against the
   image's own repo).

## Output format

Produce a single Markdown report with one section per step above. Each
section must include:

- The exact command you ran.
- The exit code.
- The stdout/stderr verbatim, in a fenced code block.

End with a one-line verdict: either `VERDICT: gh auth OK` or
`VERDICT: gh auth FAILED — <short reason>`.
