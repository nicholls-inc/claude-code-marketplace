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
 * Behaviour: build the module then exit 0 if every #guard / decide tactic
 * passed. The lean-runner script in the Docker image accepts a "test"
 * subcommand and is responsible for selecting the test target.
 *
 * Sub-phase 3b-β scope decision: /drt-oracle uses an external Python harness
 * driving `lean_run` against per-def runner files rather than `lake test`, so
 * this tool stays as the compile-time `#guard` path. Useful for in-skill
 * fixture sanity checks; not the random-input fuzzing surface.
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
