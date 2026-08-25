#!/usr/bin/env node
// tier-gate.mjs — deterministic CI gate enforcing the assurance tier map.
//
// Inputs (env):
//   PR_BODY           - full pull request description text
//   PR_LABELS         - comma-separated label list (e.g. "tier:2,needs-review")
//   CHANGED_FILES     - newline-separated list of changed file paths (repo-relative)
//   BASE_REF          - the PR's base branch/commit (accepted for interface
//                        completeness and included in the pass summary; the
//                        actual diff is supplied via CHANGED_FILES)
//   CROSSCHECK_PROTECTED_RULES - optional override path to the protected-surfaces
//                        rules file (default: .claude/rules/protected-surfaces.md)
//
// No dependencies. Node ESM. Exits 0 on pass, 1 on failure.

import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, sep } from 'node:path';
import { execSync } from 'node:child_process';

const CWD = process.cwd();
const DEFAULT_RULES_PATH = '.claude/rules/protected-surfaces.md';
const GATE_DOC = 'tier-layer-gate.md';

function readEnvList(name, sep_) {
  const raw = process.env[name] || '';
  return raw
    .split(sep_)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

function globToRegExp(glob) {
  const parts = glob.split('/').map((part) => {
    if (part === '**') return '.*';
    const escaped = part.replace(/[.+^${}()|[\]\\]/g, '\\$&');
    return escaped.replace(/\*/g, '[^/]*');
  });
  return new RegExp(`^${parts.join('/')}$`);
}

function loadProtectedGlobs(rulesPath) {
  if (!existsSync(rulesPath)) {
    return { globs: [], missing: true };
  }
  const text = readFileSync(rulesPath, 'utf8');
  const headingIdx = text.indexOf('## Machine-readable path list');
  if (headingIdx === -1) return { globs: [], missing: true };
  const afterHeading = text.slice(headingIdx);
  const fenceMatch = afterHeading.match(/```\n([\s\S]*?)```/);
  if (!fenceMatch) return { globs: [], missing: true };
  const globs = fenceMatch[1]
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l.length > 0);
  return { globs, missing: false };
}

function matchesAnyGlob(filePath, regexes) {
  return regexes.some((re) => re.test(filePath));
}

function parseDeclaredTier(prBody, prLabels) {
  const bodyMatch = (prBody || '').match(/Tier:\s*([123])\b/i);
  if (bodyMatch) return Number(bodyMatch[1]);
  const labelMatch = prLabels.find((l) => /^tier:([123])$/i.test(l));
  if (labelMatch) return Number(labelMatch.match(/^tier:([123])$/i)[1]);
  return undefined;
}

function referencedPathExists(prBody, keyword) {
  const re = new RegExp(`${keyword}:\\s*(\\S+)`, 'i');
  const match = (prBody || '').match(re);
  if (!match) return false;
  return existsSync(join(CWD, match[1]));
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

function findAttestationFile(allFiles) {
  return allFiles.find((f) => f.replace(/\\/g, '/').endsWith('.assurance/intent-check-attestation.json'));
}

function findGovernanceNoteFiles(allFiles) {
  return allFiles.filter((f) => {
    const norm = f.replace(/\\/g, '/');
    return (
      /\/\.assurance\/protected-surface-amend\//.test(norm) ||
      /\/\.assurance\/add-session-[^/]*\//.test(norm) ||
      /^\.assurance\/protected-surface-amend\//.test(norm) ||
      /^\.assurance\/add-session-[^/]*\//.test(norm)
    ) && norm.endsWith('.md');
  });
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

function printFailure(missingItems) {
  const link = gateDocLink();
  console.log('**Action needed: declare the correct tier and add missing artefacts**');
  console.log(
    `You are being asked to declare this pull request's assurance tier and supply the artefacts that tier requires because the change does not yet satisfy the tier-layer gate. Approving means the pull request carries an auditable record matching its actual risk level; declining means the change stays blocked until the tier and artefacts are corrected. Full explanation: ${link}.`
  );
  console.log('');
  for (const item of missingItems) {
    console.log(`- ${item}`);
  }
}

function main() {
  const prBody = process.env.PR_BODY || '';
  const prLabels = readEnvList('PR_LABELS', ',');
  const changedFiles = readEnvList('CHANGED_FILES', '\n');
  const baseRef = process.env.BASE_REF || '(unspecified)';
  const rulesPath = process.env.CROSSCHECK_PROTECTED_RULES || DEFAULT_RULES_PATH;

  const { globs: protectedGlobs, missing: rulesMissing } = loadProtectedGlobs(rulesPath);
  const protectedRegexes = protectedGlobs.map(globToRegExp);

  const protectedMatches = changedFiles.filter((f) => matchesAnyGlob(f, protectedRegexes));
  const floorTier = protectedMatches.length > 0 ? 3 : undefined;
  const declaredTier = parseDeclaredTier(prBody, prLabels);

  const missing = [];

  if (rulesMissing) {
    missing.push(
      `Could not read the protected-surface glob list from "${rulesPath}" (expected a "## Machine-readable path list" fenced block).`
    );
  }

  if (floorTier === undefined && declaredTier === undefined) {
    missing.push(
      'Declare this pull request\'s tier via a "Tier: N" line in the PR body or a "tier:N" label.'
    );
    printFailure(missing);
    process.exit(1);
  }

  if (floorTier !== undefined && declaredTier !== undefined && declaredTier < floorTier) {
    missing.push(
      `Protected file(s) in this diff force a floor of Tier ${floorTier}, but the PR declares Tier ${declaredTier}: ${protectedMatches.join(', ')}`
    );
    printFailure(missing);
    process.exit(1);
  }

  const effectiveTier = floorTier !== undefined ? floorTier : declaredTier;

  if (effectiveTier === 1) {
    const changedIntentFile = changedFiles.find((f) => {
      if (!f.startsWith('intent/')) return false;
      const base = f.split('/').pop();
      return base !== 'README.md' && base !== 'TEMPLATE.md';
    });
    const referenced = referencedPathExists(prBody, 'Intent');
    if (!changedIntentFile && !referenced) {
      missing.push(
        'Tier 1 requires an intent artefact: a changed file under intent/ (other than README.md or TEMPLATE.md), or an "Intent: <path>" reference in the PR body pointing to an existing file.'
      );
    }
  } else if (effectiveTier === 2) {
    const specAtRoot = existsSync(join(CWD, 'spec.md'));
    const referenced = referencedPathExists(prBody, 'Spec');
    if (!specAtRoot && !referenced) {
      missing.push(
        'Tier 2 requires a spec artefact: spec.md at the repository root, or a "Spec: <path>" reference in the PR body pointing to an existing file.'
      );
    }
  } else if (effectiveTier === 3) {
    const planAtRoot = existsSync(join(CWD, 'plan.md'));
    const planReferenced = referencedPathExists(prBody, 'Plan');
    if (!planAtRoot && !planReferenced) {
      missing.push(
        'Tier 3 requires a build plan: plan.md at the repository root, or a "Plan: <path>" reference in the PR body pointing to an existing file.'
      );
    }

    const allFiles = walk(CWD).map((f) => relative(CWD, f).split(sep).join('/'));

    if (protectedMatches.length > 0) {
      const governanceFiles = findGovernanceNoteFiles(allFiles);
      const governanceText = governanceFiles.map((f) => readFileSync(join(CWD, f), 'utf8')).join('\n');
      const unnamed = protectedMatches.filter((f) => !governanceText.includes(f));
      if (governanceFiles.length === 0 || unnamed.length > 0) {
        missing.push(
          `Tier 3 requires a governance-note block under .assurance/protected-surface-amend/ or .assurance/add-session-*/ naming each changed protected file. Not named: ${unnamed.join(', ') || protectedMatches.join(', ')}`
        );
      }
    }

    const attestation = findAttestationFile(allFiles);
    if (!attestation) {
      missing.push(
        'Tier 3 requires an intent-check attestation file matching **/.assurance/intent-check-attestation.json.'
      );
    }
  }

  if (missing.length > 0) {
    printFailure(missing);
    process.exit(1);
  }

  console.log(
    `tier-gate: PASS — Tier ${effectiveTier} artefacts present (declared: ${declaredTier ?? 'undeclared'}, floor: ${floorTier ?? 'none'}, base: ${baseRef}).`
  );
  process.exit(0);
}

main();
