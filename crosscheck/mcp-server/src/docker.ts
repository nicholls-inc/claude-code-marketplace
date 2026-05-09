import { spawn } from "node:child_process";

const DEFAULT_TIMEOUT_MS = 120_000; // 120 seconds (Dafny default)
const LEAN_TIMEOUT_MS = 240_000; // 240 seconds — Lean lake builds run longer even with pre-warmed Mathlib

interface DockerResult {
  exitCode: number;
  stdout: string;
  stderr: string;
  timedOut: boolean;
}

interface DockerOptions {
  memory?: string;
  cpus?: string;
  timeoutMs?: number;
  network?: string;
}

export function getDockerImage(): string {
  return process.env.DAFNY_DOCKER_IMAGE || "crosscheck-dafny:latest";
}

export function getLeanDockerImage(): string {
  return process.env.LEAN_DOCKER_IMAGE || "crosscheck-lean:latest";
}

function runDocker(
  image: string,
  tempDir: string,
  args: string[],
  opts: DockerOptions
): Promise<DockerResult> {
  const memory = opts.memory ?? "512m";
  const cpus = opts.cpus ?? "1";
  const timeoutMs = opts.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const network = opts.network ?? "none";

  const dockerArgs = [
    "run",
    "--rm",
    `--network=${network}`,
    `--memory=${memory}`,
    `--cpus=${cpus}`,
    "-v",
    `${tempDir}:/work`,
    image,
    ...args,
  ];

  return new Promise((resolve) => {
    const proc = spawn("docker", dockerArgs);

    let stdout = "";
    let stderr = "";
    let timedOut = false;

    const timer = setTimeout(() => {
      timedOut = true;
      proc.kill("SIGKILL");
    }, timeoutMs);

    proc.stdout.on("data", (data: Buffer) => {
      stdout += data.toString();
    });

    proc.stderr.on("data", (data: Buffer) => {
      stderr += data.toString();
    });

    proc.on("close", (code) => {
      clearTimeout(timer);
      resolve({
        exitCode: code ?? 1,
        stdout,
        stderr,
        timedOut,
      });
    });

    proc.on("error", (err) => {
      clearTimeout(timer);
      resolve({
        exitCode: 1,
        stdout,
        stderr: stderr + "\n" + err.message,
        timedOut: false,
      });
    });
  });
}

export async function runDafny(
  tempDir: string,
  args: string[]
): Promise<DockerResult> {
  return runDocker(getDockerImage(), tempDir, args, {
    memory: "512m",
    cpus: "1",
    timeoutMs: DEFAULT_TIMEOUT_MS,
    network: "none",
  });
}

export async function runLean(
  tempDir: string,
  args: string[]
): Promise<DockerResult> {
  // Lean needs more memory than Dafny: Mathlib oleans are large, and lake build
  // can spike. 2 GB is a pragmatic upper bound for small user files in a
  // pre-warmed image; tune via env if a host runs into OOM.
  const memory = process.env.LEAN_DOCKER_MEMORY || "2g";
  const cpus = process.env.LEAN_DOCKER_CPUS || "2";
  return runDocker(getLeanDockerImage(), tempDir, args, {
    memory,
    cpus,
    timeoutMs: LEAN_TIMEOUT_MS,
    network: "none",
  });
}
