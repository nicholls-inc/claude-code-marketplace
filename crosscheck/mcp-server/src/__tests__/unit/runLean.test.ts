import { describe, it, expect, vi, beforeEach } from "vitest";
import { EventEmitter } from "node:events";
import { Readable } from "node:stream";
import { spawn } from "node:child_process";

vi.mock("node:child_process");

import { runLean } from "../../docker.js";

function createMockProcess() {
  const proc = new EventEmitter() as any;
  proc.stdout = new Readable({ read() {} });
  proc.stderr = new Readable({ read() {} });
  proc.kill = vi.fn();
  return proc;
}

describe("runLean", () => {
  beforeEach(() => {
    delete process.env.LEAN_DOCKER_IMAGE;
    delete process.env.LEAN_DOCKER_MEMORY;
    delete process.env.LEAN_DOCKER_CPUS;
  });

  it("constructs docker args with Lean image, 2g memory, 2 cpus, network=none", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runLean("/tmp/workdir", ["check", "/work/program.lean"]);
    mockProc.emit("close", 0);
    await promise;

    expect(spawn).toHaveBeenCalledWith("docker", [
      "run",
      "--rm",
      "--network=none",
      "--memory=2g",
      "--cpus=2",
      "-v",
      "/tmp/workdir:/work",
      "crosscheck-lean:latest",
      "check",
      "/work/program.lean",
    ]);
  });

  it("honours LEAN_DOCKER_IMAGE / MEMORY / CPUS env overrides", async () => {
    process.env.LEAN_DOCKER_IMAGE = "custom-lean:dev";
    process.env.LEAN_DOCKER_MEMORY = "4g";
    process.env.LEAN_DOCKER_CPUS = "4";

    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runLean("/tmp/workdir", ["run", "/work/program.lean"]);
    mockProc.emit("close", 0);
    await promise;

    expect(spawn).toHaveBeenCalledWith("docker", [
      "run",
      "--rm",
      "--network=none",
      "--memory=4g",
      "--cpus=4",
      "-v",
      "/tmp/workdir:/work",
      "custom-lean:dev",
      "run",
      "/work/program.lean",
    ]);
  });

  it("times out at 240s and SIGKILLs the process", async () => {
    vi.useFakeTimers();
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runLean("/tmp/workdir", ["check", "/work/program.lean"]);

    vi.advanceTimersByTime(240_001);

    expect(mockProc.kill).toHaveBeenCalledWith("SIGKILL");

    mockProc.emit("close", null);
    const result = await promise;

    expect(result.timedOut).toBe(true);
    expect(result.exitCode).toBe(1);

    vi.useRealTimers();
  });
});
