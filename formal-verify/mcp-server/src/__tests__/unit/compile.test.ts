import { describe, it, expect, vi, beforeEach } from "vitest";
import { writeFile, readdir, readFile } from "node:fs/promises";
import { runDafny } from "../../docker.js";
import { createTempDir, removeTempDir } from "../../tempdir.js";

vi.mock("node:fs/promises");
vi.mock("../../docker.js");
vi.mock("../../tempdir.js");

import { dafnyCompile } from "../../tools/compile.js";

beforeEach(() => {
  vi.mocked(createTempDir).mockResolvedValue("/tmp/dafny-test-dir");
  vi.mocked(removeTempDir).mockResolvedValue(undefined);
  vi.mocked(writeFile).mockResolvedValue(undefined);
});

describe("dafnyCompile", () => {
  it("writes source to tempdir/program.dfy", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    await dafnyCompile({ source: "method Main() {}", target: "py" });

    expect(writeFile).toHaveBeenCalledWith(
      "/tmp/dafny-test-dir/program.dfy",
      "method Main() {}",
      "utf-8"
    );
  });

  it("calls runDafny with build args for py target", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    await dafnyCompile({ source: "method Main() {}", target: "py" });

    expect(runDafny).toHaveBeenCalledWith("/tmp/dafny-test-dir", [
      "translate",
      "py",
      "/work/program.dfy",
    ]);
  });

  it("calls runDafny with translate args for go target", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    await dafnyCompile({ source: "method Main() {}", target: "go" });

    expect(runDafny).toHaveBeenCalledWith("/tmp/dafny-test-dir", [
      "translate",
      "go",
      "/work/program.dfy",
    ]);
  });

  it("handles timeout", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 1,
      stdout: "",
      stderr: "",
      timedOut: true,
    });

    const result = await dafnyCompile({
      source: "method Main() {}",
      target: "py",
    });

    expect(result.success).toBe(false);
    expect(result.errors).toEqual([
      "Compilation timed out after 120 seconds",
    ]);
    expect(result.files).toEqual([]);
  });

  it("handles compilation failure with error lines", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 1,
      stdout: "program.dfy(2,0): Error: invalid syntax\nother output",
      stderr: "",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    const result = await dafnyCompile({
      source: "bad source",
      target: "py",
    });

    expect(result.success).toBe(false);
    expect(result.errors).toContain(
      "program.dfy(2,0): Error: invalid syntax"
    );
    expect(result.files).toEqual([]);
  });

  it("handles compilation failure without error lines (generic message)", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 1,
      stdout: "some output without matching keyword",
      stderr: "",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    const result = await dafnyCompile({
      source: "bad source",
      target: "py",
    });

    expect(result.success).toBe(false);
    expect(result.errors).toEqual([
      "Compilation failed. See rawOutput for details.",
    ]);
  });

  it("on success collects files, filters .dfy, and strips boilerplate", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });

    // Mock readdir for collectFiles - top level of tempDir
    vi.mocked(readdir).mockResolvedValue([
      {
        name: "program.dfy",
        isDirectory: () => false,
        isFile: () => true,
      },
      {
        name: "program.py",
        isDirectory: () => false,
        isFile: () => true,
      },
      {
        name: "output.py",
        isDirectory: () => false,
        isFile: () => true,
      },
    ] as any);

    vi.mocked(readFile)
      .mockResolvedValueOnce("# generated python from program\nprint('hello')\n")
      .mockResolvedValueOnce("# generated python output\nresult = 42\n");

    const result = await dafnyCompile({
      source: "method Main() {}",
      target: "py",
    });

    expect(result.success).toBe(true);
    expect(result.errors).toEqual([]);
    // .dfy files should be filtered out
    expect(result.files.every((f) => !f.path.endsWith(".dfy"))).toBe(true);
    // Should have the .py files
    expect(result.files.length).toBe(2);
    expect(result.files[0].path).toBe("program.py");
    expect(result.files[1].path).toBe("output.py");
  });

  it("excludes boilerplate files for python target", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "",
      stderr: "",
      timedOut: false,
    });

    vi.mocked(readdir).mockResolvedValue([
      {
        name: "_dafny.py",
        isDirectory: () => false,
        isFile: () => true,
      },
      {
        name: "program.py",
        isDirectory: () => false,
        isFile: () => true,
      },
    ] as any);

    vi.mocked(readFile).mockResolvedValueOnce("real code\n");

    const result = await dafnyCompile({
      source: "method Main() {}",
      target: "py",
    });

    expect(result.success).toBe(true);
    // _dafny.py should be excluded
    expect(result.files.length).toBe(1);
    expect(result.files[0].path).toBe("program.py");
  });

  it("includes rawOutput in result", async () => {
    vi.mocked(runDafny).mockResolvedValue({
      exitCode: 0,
      stdout: "build stdout",
      stderr: "build stderr",
      timedOut: false,
    });
    vi.mocked(readdir).mockResolvedValue([] as any);

    const result = await dafnyCompile({
      source: "method Main() {}",
      target: "py",
    });

    expect(result.rawOutput).toBe("build stdout\nbuild stderr");
  });
});
