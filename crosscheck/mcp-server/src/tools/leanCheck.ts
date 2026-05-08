import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { runLean } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";

interface LeanCheckInput {
  source: string;
}

export type LeanFailureKind =
  | "success"
  | "parse-error"
  | "typecheck-error"
  | "build-error"
  | "timeout";

interface LeanCheckOutput {
  success: boolean;
  kind: LeanFailureKind;
  errors: string[];
  warnings: string[];
  sorries: string[];
  rawOutput: string;
}

interface ParsedLeanOutput {
  errors: string[];
  warnings: string[];
  sorries: string[];
}

/**
 * Parse `lake build` output into errors / warnings / sorry sites.
 *
 * `lake build` reports each diagnostic as a line of the form
 *   `Crosscheck/Program.lean:LL:CC: error: <message>`
 * or with `warning:` / `info:` in place of `error:`. A `sorry` placeholder
 * surfaces as a *warning* tagged "declaration uses 'sorry'" — those are
 * expected for spec stubs and must NOT be classified as failure for
 * /lean-spec's hard gate.
 */
export function parseLeanOutput(
  stdout: string,
  stderr: string
): ParsedLeanOutput {
  const combined = stdout + "\n" + stderr;
  const lines = combined.split("\n").filter((l) => l.trim());

  const errors: string[] = [];
  const warnings: string[] = [];
  const sorries: string[] = [];

  for (const line of lines) {
    const trimmed = line.trim();

    // Strip ANSI colour codes that lake/lean sometimes emits.
    // ESC = U+001B (encoded as a string literal so the source has no control char).
    const ESC = String.fromCharCode(27);
    const clean = trimmed.replace(new RegExp(`${ESC}\\[[0-9;]*m`, "g"), "");

    if (/declaration uses 'sorry'/i.test(clean)) {
      sorries.push(clean);
      continue;
    }

    if (/\berror:\s/i.test(clean)) {
      errors.push(clean);
    } else if (/\bwarning:\s/i.test(clean)) {
      warnings.push(clean);
    }
  }

  return { errors, warnings, sorries };
}

/**
 * Classify a parsed lake-build result. Order matters: parse errors mask
 * everything downstream, so check parse-shaped messages first.
 */
export function classifyLeanFailure(
  exitCode: number,
  errors: string[]
): LeanFailureKind {
  if (exitCode === 0 && errors.length === 0) {
    return "success";
  }

  const hasParse = errors.some((e) =>
    /(unexpected token|expected|unknown identifier|invalid syntax)/i.test(e)
  );
  if (hasParse) return "parse-error";

  const hasType = errors.some((e) =>
    /(type mismatch|expected type|application type mismatch|failed to synthesize)/i.test(
      e
    )
  );
  if (hasType) return "typecheck-error";

  return "build-error";
}

export async function leanCheck(
  input: LeanCheckInput
): Promise<LeanCheckOutput> {
  const tempDir = await createTempDir("lean-");

  try {
    const programPath = join(tempDir, "program.lean");
    await writeFile(programPath, input.source, "utf-8");

    const result = await runLean(tempDir, ["check", "/work/program.lean"]);

    if (result.timedOut) {
      return {
        success: false,
        kind: "timeout",
        errors: ["Lean check timed out after 240 seconds"],
        warnings: [],
        sorries: [],
        rawOutput: (result.stdout + "\n" + result.stderr).trim(),
      };
    }

    const { errors, warnings, sorries } = parseLeanOutput(
      result.stdout,
      result.stderr
    );
    const kind = classifyLeanFailure(result.exitCode, errors);
    const success = kind === "success";
    const rawOutput = (result.stdout + "\n" + result.stderr).trim();

    return {
      success,
      kind,
      errors,
      warnings,
      sorries,
      rawOutput,
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
