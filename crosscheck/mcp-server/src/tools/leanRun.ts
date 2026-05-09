import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { runLean } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";

interface LeanRunInput {
  source: string;
}

interface LeanRunOutput {
  success: boolean;
  exitCode: number;
  stdout: string;
  stderr: string;
  timedOut: boolean;
}

/**
 * Compile and execute a Lean file's `main : IO Unit` entry point.
 *
 * Used by /lean-impl for sanity-checking a functional model against
 * worked-example inputs from the informal spec, and by /drt-oracle as the
 * Lean-side runner that the external Python harness invokes per random
 * input. /lean-spec does NOT call this — spec stubs contain `sorry` and are
 * not meant to run.
 */
export async function leanRun(input: LeanRunInput): Promise<LeanRunOutput> {
  const tempDir = await createTempDir("lean-");

  try {
    const programPath = join(tempDir, "program.lean");
    await writeFile(programPath, input.source, "utf-8");

    const result = await runLean(tempDir, ["run", "/work/program.lean"]);

    return {
      success: !result.timedOut && result.exitCode === 0,
      exitCode: result.exitCode,
      stdout: result.stdout.trim(),
      stderr: result.stderr.trim(),
      timedOut: result.timedOut,
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
