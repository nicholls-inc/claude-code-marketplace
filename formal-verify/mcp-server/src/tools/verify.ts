import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { runDafny } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";

interface VerifyInput {
  source: string;
}

interface VerifyOutput {
  success: boolean;
  errors: string[];
  warnings: string[];
  rawOutput: string;
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
      };
    }

    const { errors, warnings } = parseDafnyOutput(result.stdout, result.stderr);
    const success = result.exitCode === 0 && errors.length === 0;

    return {
      success,
      errors,
      warnings,
      rawOutput: (result.stdout + "\n" + result.stderr).trim(),
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
