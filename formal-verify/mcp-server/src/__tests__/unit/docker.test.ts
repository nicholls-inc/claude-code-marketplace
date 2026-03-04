import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { EventEmitter } from "node:events";
import { Readable } from "node:stream";
import { spawn } from "node:child_process";

vi.mock("node:child_process");

import { runDafny, getDockerImage } from "../../docker.js";

function createMockProcess() {
  const proc = new EventEmitter() as any;
  proc.stdout = new Readable({ read() {} });
  proc.stderr = new Readable({ read() {} });
  proc.kill = vi.fn();
  return proc;
}

describe("getDockerImage", () => {
  const origEnv = process.env.DAFNY_DOCKER_IMAGE;

  afterEach(() => {
    if (origEnv === undefined) {
      delete process.env.DAFNY_DOCKER_IMAGE;
    } else {
      process.env.DAFNY_DOCKER_IMAGE = origEnv;
    }
  });

  it("returns env var when set", () => {
    process.env.DAFNY_DOCKER_IMAGE = "custom-image:v2";
    expect(getDockerImage()).toBe("custom-image:v2");
  });

  it("returns default image when env var is not set", () => {
    delete process.env.DAFNY_DOCKER_IMAGE;
    expect(getDockerImage()).toBe("formal-verify-dafny:latest");
  });
});

describe("runDafny", () => {
  beforeEach(() => {
    delete process.env.DAFNY_DOCKER_IMAGE;
  });

  it("constructs correct docker args", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    // Emit close immediately
    mockProc.emit("close", 0);

    await promise;

    expect(spawn).toHaveBeenCalledWith("docker", [
      "run",
      "--rm",
      "--network=none",
      "--memory=512m",
      "--cpus=1",
      "-v",
      "/tmp/workdir:/work",
      "formal-verify-dafny:latest",
      "verify",
      "/work/program.dfy",
    ]);
  });

  it("collects stdout and stderr", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    mockProc.stdout.emit("data", Buffer.from("verification output"));
    mockProc.stderr.emit("data", Buffer.from("some warning"));
    mockProc.emit("close", 0);

    const result = await promise;

    expect(result.stdout).toBe("verification output");
    expect(result.stderr).toBe("some warning");
  });

  it("resolves with exit code, stdout, stderr, and timedOut false on close", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    mockProc.stdout.emit("data", Buffer.from("ok"));
    mockProc.emit("close", 0);

    const result = await promise;

    expect(result).toEqual({
      exitCode: 0,
      stdout: "ok",
      stderr: "",
      timedOut: false,
    });
  });

  it("defaults null exit code to 1", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);
    mockProc.emit("close", null);

    const result = await promise;

    expect(result.exitCode).toBe(1);
  });

  it("sends SIGKILL and sets timedOut on timeout", async () => {
    vi.useFakeTimers();

    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    // Advance past the 120s timeout
    vi.advanceTimersByTime(120_001);

    expect(mockProc.kill).toHaveBeenCalledWith("SIGKILL");

    // Process closes after being killed
    mockProc.emit("close", null);

    const result = await promise;

    expect(result.timedOut).toBe(true);
    expect(result.exitCode).toBe(1);

    vi.useRealTimers();
  });

  it("resolves with error info on spawn error", async () => {
    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    mockProc.emit("error", new Error("spawn ENOENT"));

    const result = await promise;

    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain("spawn ENOENT");
    expect(result.timedOut).toBe(false);
  });

  it("clears timer on close", async () => {
    vi.useFakeTimers();
    const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout");

    const mockProc = createMockProcess();
    vi.mocked(spawn).mockReturnValue(mockProc as any);

    const promise = runDafny("/tmp/workdir", ["verify", "/work/program.dfy"]);

    mockProc.emit("close", 0);

    await promise;

    expect(clearTimeoutSpy).toHaveBeenCalled();

    vi.useRealTimers();
  });
});
