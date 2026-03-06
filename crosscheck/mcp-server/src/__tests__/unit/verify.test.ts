import { describe, it, expect, vi, beforeEach } from "vitest";
import { writeFile } from "node:fs/promises";
import { runDafny } from "../../docker.js";
import { createTempDir } from "../../tempdir.js";

vi.mock("node:fs/promises");
vi.mock("../../docker.js");
vi.mock("../../tempdir.js");

import { dafnyVerify } from "../../tools/verify.js";

beforeEach(() => {
  vi.mocked(createTempDir).mockResolvedValue("/tmp/dafny-test-dir");
  vi.mocked(writeFile).mockResolvedValue(undefined);
});

describe("dafnyVerify", () => {
  it("writes source to tempdir/program.dfy", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });

    await dafnyVerify({ source: "method Main() {}" });

    expect(writeFile).toHaveBeenCalledWith(
      "/tmp/dafny-test-dir/program.dfy",
      "method Main() {}",
      "utf-8"
    );
  });

  it("calls runDafny with verify args", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });

    await dafnyVerify({ source: "method Main() {}" });

    expect(runDafny).toHaveBeenCalledWith("/tmp/dafny-test-dir", [
      "verify",
      "/work/program.dfy",
    ]);
  });

  it("returns success when exitCode is 0 and no errors parsed", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "Dafny program verifier finished with 1 verified, 0 errors",
      stderr: "",
      timedOut: false,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.success).toBe(true);
    expect(result.errors).toEqual([]);
    expect(result).toHaveProperty("difficulty");
    expect(result.difficulty).toEqual(
      expect.objectContaining({
        proofHintCount: expect.any(Number),
        emptyLemmaBodyCount: expect.any(Number),
        trivialProof: expect.any(Boolean),
      })
    );
  });

  it("returns failure when exitCode is 0 but errors are parsed", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "program.dfy(3,4): Error: assertion might not hold",
      stderr: "",
      timedOut: false,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.success).toBe(false);
    expect(result.errors).toContain(
      "program.dfy(3,4): Error: assertion might not hold"
    );
  });

  it("returns failure when exitCode is non-zero", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 1,
      stdout: "",
      stderr: "Error: something failed",
      timedOut: false,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.success).toBe(false);
  });

  it("handles timeout", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 1,
      stdout: "",
      stderr: "",
      timedOut: true,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.success).toBe(false);
    expect(result.errors).toEqual([
      "Verification timed out after 120 seconds",
    ]);
    expect(result.difficulty).toEqual({
      solverTimeMs: null,
      resourceCount: null,
      proofHintCount: 0,
      emptyLemmaBodyCount: 0,
      trivialProof: false,
    });
  });

  it("includes raw output from stdout and stderr", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "stdout content",
      stderr: "stderr content",
      timedOut: false,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.rawOutput).toBe("stdout content\nstderr content");
  });

  it("includes warnings in output", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "program.dfy(1,0): Warning: unused variable",
      stderr: "",
      timedOut: false,
    });

    const result = await dafnyVerify({ source: "method Main() {}" });

    expect(result.warnings).toContain(
      "program.dfy(1,0): Warning: unused variable"
    );
  });

});
