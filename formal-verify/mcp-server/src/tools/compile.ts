import { writeFile, readdir, readFile } from "node:fs/promises";
import { join, basename } from "node:path";
import { runDafny } from "../docker.js";
import { createTempDir, removeTempDir } from "../tempdir.js";

type Target = "py" | "go";

interface CompileInput {
  source: string;
  target: Target;
}

interface OutputFile {
  path: string;
  content: string;
}

interface CompileOutput {
  success: boolean;
  errors: string[];
  files: OutputFile[];
  rawOutput: string;
}

const PYTHON_EXCLUDE_FILES = ["_dafny.py", "__pycache__"];
const PYTHON_STRIP_PATTERNS = [
  /^from _dafny import.*$/gm,
  /^import _dafny.*$/gm,
  /^import _System.*$/gm,
  /^from _System import.*$/gm,
];

const GO_EXCLUDE_FILES = ["dafny.go", "System_.go"];
const GO_EXCLUDE_DIRS = ["dafny", "System_"];
const GO_STRIP_PATTERNS = [
  /^\s*_dafny\s+"[^"]*dafny".*$/gm,
  /^\s*_System\s+"[^"]*System_".*$/gm,
];

function stripBoilerplate(content: string, target: Target): string {
  const patterns = target === "py" ? PYTHON_STRIP_PATTERNS : GO_STRIP_PATTERNS;
  let result = content;
  for (const pattern of patterns) {
    result = result.replace(pattern, "");
  }
  // Clean up consecutive blank lines left by stripping
  result = result.replace(/\n{3,}/g, "\n\n");
  return result.trim() + "\n";
}

function shouldExclude(filePath: string, target: Target): boolean {
  const name = basename(filePath);

  if (target === "py") {
    return PYTHON_EXCLUDE_FILES.some(
      (ex) => name === ex || filePath.includes(ex)
    );
  }

  if (target === "go") {
    if (GO_EXCLUDE_FILES.includes(name)) return true;
    return GO_EXCLUDE_DIRS.some((dir) => filePath.includes(`/${dir}/`));
  }

  return false;
}

async function collectFiles(
  dir: string,
  base: string,
  target: Target
): Promise<OutputFile[]> {
  const files: OutputFile[] = [];
  const entries = await readdir(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = join(dir, entry.name);
    const relativePath = join(base, entry.name);

    if (entry.isDirectory()) {
      const subFiles = await collectFiles(fullPath, relativePath, target);
      files.push(...subFiles);
    } else if (entry.isFile()) {
      const ext = target === "py" ? ".py" : ".go";
      if (!entry.name.endsWith(ext)) continue;
      if (shouldExclude(relativePath, target)) continue;

      const content = await readFile(fullPath, "utf-8");
      files.push({
        path: relativePath,
        content: stripBoilerplate(content, target),
      });
    }
  }

  return files;
}

export async function dafnyCompile(
  input: CompileInput
): Promise<CompileOutput> {
  const tempDir = await createTempDir();

  try {
    const programPath = join(tempDir, "program.dfy");
    await writeFile(programPath, input.source, "utf-8");

    const targetFlag = input.target === "py" ? "py" : "go";
    const result = await runDafny(tempDir, [
      "build",
      `--target:${targetFlag}`,
      "/work/program.dfy",
    ]);

    if (result.timedOut) {
      return {
        success: false,
        errors: ["Compilation timed out after 120 seconds"],
        files: [],
        rawOutput: result.stdout + "\n" + result.stderr,
      };
    }

    if (result.exitCode !== 0) {
      const errorLines = (result.stdout + "\n" + result.stderr)
        .split("\n")
        .filter((l) => /Error/i.test(l))
        .map((l) => l.trim());

      return {
        success: false,
        errors:
          errorLines.length > 0
            ? errorLines
            : ["Compilation failed. See rawOutput for details."],
        files: [],
        rawOutput: (result.stdout + "\n" + result.stderr).trim(),
      };
    }

    // Collect output files from the temp directory
    const files = await collectFiles(tempDir, "", input.target);

    // Exclude the original .dfy file from results
    const outputFiles = files.filter((f) => !f.path.endsWith(".dfy"));

    return {
      success: true,
      errors: [],
      files: outputFiles,
      rawOutput: (result.stdout + "\n" + result.stderr).trim(),
    };
  } finally {
    await removeTempDir(tempDir);
  }
}
