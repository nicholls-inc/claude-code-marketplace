import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { runDafny } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";

interface VerifyInput {
  source: string;
}

export interface DifficultyMetrics {
  solverTimeMs: number | null;
  resourceCount: number | null;
  proofHintCount: number;
  emptyLemmaBodyCount: number;
  trivialProof: boolean;
}

interface VerifyOutput {
  success: boolean;
  errors: string[];
  warnings: string[];
  rawOutput: string;
  difficulty: DifficultyMetrics;
}

export function extractDifficultyMetrics(
  source: string,
  rawOutput: string
): DifficultyMetrics {
  // Parse solver time from patterns like "finished in X.XXs"
  let solverTimeMs: number | null = null;
  const timeMatch = rawOutput.match(/finished\s+in\s+(\d+(?:\.\d+)?)s/);
  if (timeMatch) {
    solverTimeMs = Math.round(parseFloat(timeMatch[1]) * 1000);
  }

  // Parse resource count from "resource count: NNN" or similar
  let resourceCount: number | null = null;
  const resourceMatch = rawOutput.match(/resource\s+count:\s*(\d+)/i);
  if (resourceMatch) {
    resourceCount = parseInt(resourceMatch[1], 10);
  }

  // Count proof hints: lines matching assert or calc blocks
  const proofHintCount = (source.match(/^\s*(assert\s|calc\s*\{)/gm) || [])
    .length;

  // Count empty lemma bodies
  const emptyLemmaBodyCount = (
    source.match(/lemma\s+\w+[^{]*\{\s*\}/g) || []
  ).length;

  // Determine if proof is trivial
  const trivialProof =
    (proofHintCount === 0 && emptyLemmaBodyCount > 0) ||
    (proofHintCount === 0 && solverTimeMs !== null && solverTimeMs < 2000);

  return {
    solverTimeMs,
    resourceCount,
    proofHintCount,
    emptyLemmaBodyCount,
    trivialProof,
  };
}

export function parseDafnyOutput(
  stdout: string,
  stderr: string
): { errors: string[]; warnings: string[] } {
  const combined = stdout + "\n" + stderr;
  const lines = combined.split("\n").filter((l) => l.trim());
  const errors: string[] = [];
  const warnings: string[] = [];

  for (const line of lines) {
    if (/Error/i.test(line) && !/^Dafny program verifier/.test(line)) {
      errors.push(line.trim());
    } else if (/Warning/i.test(line)) {
      warnings.push(line.trim());
    }
  }

  return { errors, warnings };
}

export async function dafnyVerify(input: VerifyInput): Promise<VerifyOutput> {
  const tempDir = await createTempDir();

  try {
    const programPath = join(tempDir, "program.dfy");
    await writeFile(programPath, input.source, "utf-8");

    const result = await runDafny(tempDir, ["verify", "/work/program.dfy"]);

    if (result.timedOut) {
      return {
        success: false,
        errors: ["Verification timed out after 120 seconds"],
        warnings: [],
        rawOutput: result.stdout + "\n" + result.stderr,
        difficulty: {
          solverTimeMs: null,
          resourceCount: null,
          proofHintCount: 0,
          emptyLemmaBodyCount: 0,
          trivialProof: false,
        },
      };
    }

    const { errors, warnings } = parseDafnyOutput(result.stdout, result.stderr);
    const success = result.exitCode === 0 && errors.length === 0;
    const rawOutput = (result.stdout + "\n" + result.stderr).trim();
    const difficulty = extractDifficultyMetrics(input.source, rawOutput);

    return {
      success,
      errors,
      warnings,
      rawOutput,
      difficulty,
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
