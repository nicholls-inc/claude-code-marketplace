import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { runLean } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";
import { parseLeanOutput } from "./leanCheck.js";

interface LeanTestInput {
  source: string;
}

interface LeanTestOutput {
  success: boolean;
  exitCode: number;
  errors: string[];
  warnings: string[];
  rawOutput: string;
  timedOut: boolean;
}

/**
 * Run a Lean test harness over a user-supplied module.
 *
 * Used by /drt-oracle (sub-phase 3b-β) once the differential pipeline lands.
 * Behaviour today: build the module then exit 0 if every #guard / decide
 * tactic passed. The lean-runner script in the Docker image accepts a "test"
 * subcommand and is responsible for selecting the test target.
 */
export async function leanTest(input: LeanTestInput): Promise<LeanTestOutput> {
  const tempDir = await createTempDir("lean-");

  try {
    const programPath = join(tempDir, "program.lean");
    await writeFile(programPath, input.source, "utf-8");

    const result = await runLean(tempDir, ["test", "/work/program.lean"]);

    const { errors, warnings } = parseLeanOutput(result.stdout, result.stderr);
    const success = !result.timedOut && result.exitCode === 0 && errors.length === 0;

    return {
      success,
      exitCode: result.exitCode,
      errors,
      warnings,
      rawOutput: (result.stdout + "\n" + result.stderr).trim(),
      timedOut: result.timedOut,
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
