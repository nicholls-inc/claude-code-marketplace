# Gate: assurance-init prompts

`/assurance-init` is a Crosscheck skill that scaffolds a repository's governance
skeleton — the documents and rules that let humans and AI agents agree on which
parts of the codebase are safe to change freely and which require a
human-authorised amendment. Because it writes files that later gates depend on,
it pauses twice to check its assumptions with you before writing anything. This
document explains both pauses.

## Term used below

**Protected surface** — a file or pattern of files (for example, module
invariant specifications, or the rules file itself) that this scaffolding
marks as requiring an explicit, human-authorised amendment before it can
change. Automated agents may propose a change to a protected surface; they may
not apply one unilaterally.

## Pause 1 — the pre-flight collision prompt

**What it protects and why it exists.** Before writing anything, the skill
checks whether any of the files or directories it is about to create already
exist — for example, from a previous partial run, or from unrelated tooling
already present in the repository. Without this check, a second run of the
skill could silently overwrite governance documents someone has already
started editing, or clobber unrelated files that happen to share a path.

**What you are being asked to decide.** For each colliding path, whether to
keep what is already there, replace it, or stop the whole scaffolding run.

**What each option means:**

- **Skip** — the existing file is left completely untouched. The skill notes
  the skip in its final summary and continues scaffolding everything else.
- **Overwrite** — the existing file is replaced with the skill's generated
  version. Anything you had already written into it is lost, so only choose
  this if you are confident the existing file is stale or disposable.
- **Abort** — the skill stops immediately. Nothing is written, including
  files that had no collision at all. Use this if you are not sure what the
  colliding files are and want to check before proceeding.

An answer that is ambiguous, unclear, or not one of the three options above is
always treated as **skip** — the safest default, since it never destroys
existing content.

**How long this takes.** Usually under a minute: it is a single yes/no-style
choice per colliding path, and most runs have none.

## Pause 2 — Q1 to Q3 (the dual-track enforcement questions)

**What it protects and why it exists.** The scaffolding generates two
enforcement stubs — a local pre-commit check and a continuous-integration
(CI) check — that later become the repository's real governance gates. Both
need to know which tools the repository already uses so the generated stubs
are wired into the right place rather than into a system the repository does
not run.

**What you are being asked to decide.**

1. **Pre-commit framework** — which tool (if any) runs checks locally before
   a commit is made: `pre-commit.com`, `lefthook`, `husky`, or none.
2. **CI system** — which continuous-integration platform runs checks on every
   pull request: GitHub Actions, GitLab CI, CircleCI, or another named
   system.
3. **Seed modules** — up to three modules in the repository whose behaviour
   is most load-bearing (where a regression would have the widest
   consequences). These receive skeleton "invariant" documents — write-ups of
   the properties that module must always preserve — for a human to fill in
   afterwards.

Where the answer can be inferred confidently — from an existing configuration
file, from a prior automated audit, or from arguments you supplied when
invoking the skill — the skill pre-fills it and only asks you to confirm or
override. It only asks from scratch when nothing lets it infer an answer.

**What happens next in each case.** Your answers determine which stub files
the skill writes (for example, a `.pre-commit-config.yaml` entry versus a
GitHub Actions workflow) and which three modules receive skeleton invariant
documents. Choosing "none" for the pre-commit framework does not block the
run; the skill records that a fast local gate should be installed before the
next scaffolding step. Seed-module lists longer than three are rejected —
onboarding deliberately starts narrow.

**How long this takes.** A few minutes: two short multiple-choice answers and
one judgement call about which modules matter most.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
