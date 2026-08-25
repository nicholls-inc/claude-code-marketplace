#!/usr/bin/env node
// Deterministic PreToolUse hook: blocks edits to protected surfaces unless a
// Protected-Surface Amendment governance-note block already names the file.
//
// Reads the PreToolUse JSON payload from stdin: { tool_name, tool_input: { file_path, ... } }.
// Exit 0  -> allow the edit.
// Exit 2  -> block the edit; prints the gate message to stderr.
//
// Fail-open policy: if .claude/rules/protected-surfaces.md (or its env override)
// is missing, this hook exits 0. That is a deliberate exception to "deny by
// default" -- the rules file may not exist yet (e.g. mid-bootstrap, or a repo
// that has not adopted Package B at all), and a hook that cannot find its own
// policy input must not brick every edit in the repo. Any other failure mode
// (bad glob, missing repo root, unreadable amendment files) should still be
// treated conservatively, but an absent rules file is not a policy failure --
// it is "this repo has not opted in yet".
//
// CROSSCHECK_PROTECTED_RULES env var overrides the rules file path, for tests.

import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { join, resolve, relative, sep } from 'node:path';

function readStdin() {
  try {
    return readFileSync(0, 'utf8');
  } catch {
    return '';
  }
}

function getRepoRoot() {
  try {
    return execFileSync('git', ['rev-parse', '--show-toplevel'], {
      encoding: 'utf8',
    }).trim();
  } catch {
    return process.cwd();
  }
}

// Extract the fenced ```text/plain code block under the
// "## Machine-readable path list" heading in protected-surfaces.md.
// Returns null when the rules file is absent (fail open, see policy note),
// 'malformed' when it exists but the machine-readable block cannot be parsed
// (fail closed -- a present-but-unparseable policy must not silently disable
// enforcement), or the glob list.
function loadGlobs(rulesPath) {
  if (!existsSync(rulesPath)) return null;
  const content = readFileSync(rulesPath, 'utf8');
  const headingIdx = content.indexOf('## Machine-readable path list');
  if (headingIdx === -1) return 'malformed';
  const after = content.slice(headingIdx);
  const fenceMatch = after.match(/```[^\n]*\n([\s\S]*?)```/);
  if (!fenceMatch) return 'malformed';
  return fenceMatch[1]
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

// Simple glob matcher supporting ** (any depth, including zero segments)
// and * (any run of characters within a single path segment).
function globToRegExp(glob) {
  let re = '';
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === '*' && glob[i + 1] === '*') {
      // ** consumes any number of path segments (including none).
      // Absorb an adjacent slash on either side so "a/**/b" matches "a/b".
      let j = i + 2;
      if (glob[j] === '/') j++;
      re += '.*';
      i = j - 1;
    } else if (c === '*') {
      re += '[^/]*';
    } else if (c === '?') {
      re += '[^/]';
    } else if ('.+^${}()|[]\\'.includes(c)) {
      re += '\\' + c;
    } else {
      re += c;
    }
  }
  return new RegExp('^' + re + '$');
}

function matchesAnyGlob(relPath, globs) {
  return globs.some((g) => globToRegExp(g).test(relPath));
}

// Scan .assurance/protected-surface-amend/*.md and .assurance/add-session-*/*.md
// for a "## Protected-Surface Amendment" block whose "Target file(s)" section
// names the given (repo-relative) file path as a substring.
function hasGovernanceNote(repoRoot, relFilePath) {
  const candidateDirs = [];
  const amendDir = join(repoRoot, '.assurance', 'protected-surface-amend');
  if (existsSync(amendDir)) candidateDirs.push(amendDir);

  const assuranceRoot = join(repoRoot, '.assurance');
  if (existsSync(assuranceRoot)) {
    let entries = [];
    try {
      entries = readdirSync(assuranceRoot, { withFileTypes: true });
    } catch {
      entries = [];
    }
    for (const entry of entries) {
      if (entry.isDirectory() && entry.name.startsWith('add-session-')) {
        candidateDirs.push(join(assuranceRoot, entry.name));
      }
    }
  }

  for (const dir of candidateDirs) {
    let files = [];
    try {
      files = readdirSync(dir).filter((f) => f.endsWith('.md'));
    } catch {
      continue;
    }
    for (const file of files) {
      let content;
      try {
        content = readFileSync(join(dir, file), 'utf8');
      } catch {
        continue;
      }
      if (blockNamesFile(content, relFilePath)) return true;
    }
  }
  return false;
}

// A file's content may contain multiple "## Protected-Surface Amendment"
// blocks; check each one's "Target file(s)" section independently.
function blockNamesFile(content, relFilePath) {
  const blockRe = /## Protected-Surface Amendment[\s\S]*?(?=\n## Protected-Surface Amendment|$)/g;
  const blocks = content.match(blockRe);
  if (!blocks) return false;
  for (const block of blocks) {
    // The amendment template names the primary path on the "Target file(s)"
    // line and enumerates every further file in the "Diff Plan" table, so
    // search the whole block: any explicit mention of the path counts.
    if (block.includes(relFilePath)) return true;
  }
  return false;
}

function deriveGateLink(repoRoot) {
  const fallback = 'docs/gates/protected-surface-hook.md';
  let remoteUrl;
  try {
    remoteUrl = execFileSync('git', ['remote', 'get-url', 'origin'], {
      cwd: repoRoot,
      encoding: 'utf8',
    }).trim();
  } catch {
    return fallback;
  }

  let ownerRepo = null;
  // git@github.com:owner/repo.git
  let m = remoteUrl.match(/^git@github\.com:(.+?)(\.git)?$/);
  if (m) ownerRepo = m[1];
  if (!ownerRepo) {
    // https://github.com/owner/repo(.git)
    m = remoteUrl.match(/^https?:\/\/github\.com\/(.+?)(\.git)?\/?$/);
    if (m) ownerRepo = m[1];
  }
  if (!ownerRepo) return fallback;
  return `https://github.com/${ownerRepo}/blob/main/docs/gates/protected-surface-hook.md`;
}

function printGateMessage(link) {
  const msg = [
    '**Action needed: run /protected-surface-amend before editing**',
    `You are being asked to allow this edit to a protected surface because the file is a protected surface with no governance-note block in the working tree. Approving means generating the block via /protected-surface-amend then re-editing; declining means the file stays unchanged. Full explanation: ${link}.`,
  ].join('\n');
  process.stderr.write(msg + '\n');
}

function main() {
  const raw = readStdin();
  let payload;
  try {
    payload = JSON.parse(raw);
  } catch {
    // Malformed payload: nothing to check against, allow.
    process.exit(0);
  }

  // Edit/Write carry file_path; NotebookEdit carries notebook_path.
  const filePath =
    payload?.tool_input?.file_path ?? payload?.tool_input?.notebook_path;
  if (!filePath) process.exit(0);

  const repoRoot = getRepoRoot();
  const rulesPath =
    process.env.CROSSCHECK_PROTECTED_RULES ||
    join(repoRoot, '.claude', 'rules', 'protected-surfaces.md');

  const globs = loadGlobs(rulesPath);
  if (globs === null) {
    if (process.env.CROSSCHECK_PROTECTED_RULES) {
      // An explicit override pointing at a missing file is test plumbing gone
      // wrong, not "repo has not opted in" -- fail closed rather than letting
      // the env var act as a bypass.
      process.stderr.write(
        `protected-surface-guard: CROSSCHECK_PROTECTED_RULES points at a missing file (${rulesPath}); blocking the edit as a precaution.\n`
      );
      process.exit(2);
    }
    // Rules file missing entirely: fail open (see policy note above).
    process.exit(0);
  }
  if (globs === 'malformed') {
    // Rules file present but its machine-readable list is unparseable: fail
    // closed for every edit until the list is repaired, so a formatting change
    // cannot silently disable enforcement.
    process.stderr.write(
      `protected-surface-guard: ${rulesPath} exists but its "## Machine-readable path list" fenced block could not be parsed; blocking the edit as a precaution. Repair the block to restore normal operation.\n`
    );
    process.exit(2);
  }

  const absFilePath = resolve(repoRoot, filePath);
  let relFilePath = relative(repoRoot, absFilePath).split(sep).join('/');

  if (!matchesAnyGlob(relFilePath, globs)) {
    process.exit(0);
  }

  if (hasGovernanceNote(repoRoot, relFilePath)) {
    process.exit(0);
  }

  printGateMessage(deriveGateLink(repoRoot));
  process.exit(2);
}

main();
