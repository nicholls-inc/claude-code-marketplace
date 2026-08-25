#!/usr/bin/env node
// incident-eval-check.mjs — deterministic CI gate enforcing that every
// incident-referencing PR ships a regression eval and a candidate invariant.
//
// Inputs (env):
//   PR_BODY          - full pull request description text
//   PR_LABELS        - comma-separated label list
//   COMMIT_MESSAGES  - newline-separated commit messages for the PR
//
// No dependencies. Node ESM. Exits 0 always unless the check applies and fails
// (exit 1 in that case).

import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join, relative, sep } from 'node:path';
import { execSync } from 'node:child_process';

const CWD = process.cwd();
const GATE_DOC = 'README.md';
const INVARIANT_DIRS = ['docs/invariants', 'crosscheck/docs/invariants'];
const EVAL_DIR = 'evals';

function readEnvList(name, sep_) {
  const raw = process.env[name] || '';
  return raw
    .split(sep_)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

function walk(dir, out = []) {
  let entries;
  try {
    entries = readdirSync(dir, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === '.git') continue;
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full, out);
    } else {
      out.push(full);
    }
  }
  return out;
}

function gitOriginUrl() {
  try {
    return execSync('git remote get-url origin', { cwd: CWD }).toString().trim();
  } catch {
    return '';
  }
}

function ownerRepoFromRemote(url) {
  let m = url.match(/git@github\.com:([^/]+)\/(.+?)(\.git)?$/);
  if (m) return `${m[1]}/${m[2]}`;
  m = url.match(/https:\/\/github\.com\/([^/]+)\/([^/]+?)(\.git)?\/?$/);
  if (m) return `${m[1]}/${m[2]}`;
  return '';
}

function gateDocLink() {
  const ownerRepo = ownerRepoFromRemote(gitOriginUrl());
  if (!ownerRepo) return `docs/gates/${GATE_DOC}`;
  return `https://github.com/${ownerRepo}/blob/main/docs/gates/${GATE_DOC}`;
}

function findIncidentId(prBody, commitMessages) {
  const sources = [prBody || '', ...commitMessages];
  for (const text of sources) {
    const m = text.match(/Fixes-Incident:\s*(\S+)/i);
    if (m) return m[1].replace(/[.,;]$/, '');
  }
  return undefined;
}

function printFailure(missingItems) {
  const link = gateDocLink();
  console.log('**Action needed: add an eval and candidate invariant**');
  console.log(
    `You are being asked to add a regression eval and a candidate invariant for this incident because every production incident must leave both artefacts in the suite. Approving means the incident becomes a permanent regression check and a documented invariant; declining means the change stays blocked until both artefacts are added. Full explanation: ${link}.`
  );
  console.log('');
  for (const item of missingItems) {
    console.log(`- ${item}`);
  }
}

function main() {
  const prBody = process.env.PR_BODY || '';
  const prLabels = readEnvList('PR_LABELS', ',');
  const commitMessages = readEnvList('COMMIT_MESSAGES', '\n');

  const hasIncidentLabel = prLabels.some((l) => l.toLowerCase() === 'incident');
  const incidentId = findIncidentId(prBody, commitMessages);

  if (!incidentId && !hasIncidentLabel) {
    console.log('no incident reference — skipped');
    process.exit(0);
  }

  const missing = [];

  if (!incidentId) {
    missing.push(
      'The "incident" label is set but no incident id was found. Add a "Fixes-Incident: <id>" line to the PR body or a commit message.'
    );
    printFailure(missing);
    process.exit(1);
  }

  const allFiles = walk(CWD).map((f) => relative(CWD, f).split(sep).join('/'));

  const evalFiles = allFiles.filter((f) => f.startsWith(`${EVAL_DIR}/`));
  const evalMatch = evalFiles.find((f) => {
    if (f.includes(incidentId)) return true;
    try {
      return readFileSync(join(CWD, f), 'utf8').includes(incidentId);
    } catch {
      return false;
    }
  });
  if (!evalMatch) {
    missing.push(
      `No eval under ${EVAL_DIR}/ names or references incident "${incidentId}". Add one so this incident stays a permanent regression test.`
    );
  }

  const invariantFiles = allFiles.filter((f) =>
    INVARIANT_DIRS.some((d) => f.startsWith(`${d}/`))
  );
  const invariantMatch = invariantFiles.find((f) => {
    try {
      return readFileSync(join(CWD, f), 'utf8').includes(incidentId);
    } catch {
      return false;
    }
  });
  if (!invariantMatch) {
    missing.push(
      `No candidate invariant under docs/invariants/ or crosscheck/docs/invariants/ references incident "${incidentId}". Add or amend one to capture what the incident revealed.`
    );
  }

  if (missing.length > 0) {
    printFailure(missing);
    process.exit(1);
  }

  console.log(`incident-eval-check: PASS — incident "${incidentId}" has an eval and a candidate invariant.`);
  process.exit(0);
}

main();
