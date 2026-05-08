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
 * Used primarily by /lean-impl (sub-phase 3b-β) for sanity-checking a
 * functional model on a few inputs before /drt-oracle takes over. /lean-spec
 * does NOT call this — spec stubs contain `sorry` and are not meant to run.
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
