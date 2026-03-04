import { spawn } from "node:child_process";

const TIMEOUT_MS = 120_000; // 120 seconds

interface DockerResult {
  exitCode: number;
  stdout: string;
  stderr: string;
  timedOut: boolean;
}

export function getDockerImage(): string {
  return process.env.DAFNY_DOCKER_IMAGE || "formal-verify-dafny:latest";
}

export async function runDafny(
  tempDir: string,
  args: string[]
): Promise<DockerResult> {
  const image = getDockerImage();

  const dockerArgs = [
    "run",
    "--rm",
    "--network=none",
    "--memory=512m",
    "--cpus=1",
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
    }, TIMEOUT_MS);

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
