import { describe, it, expect, vi, beforeEach } from "vitest";
import { mkdtemp, rm, readdir, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { v4 as uuidv4 } from "uuid";

vi.mock("node:fs/promises");
vi.mock("node:os");
vi.mock("uuid");

import { createTempDir, removeTempDir, cleanupStaleDirs } from "../../tempdir.js";

beforeEach(() => {
  vi.mocked(tmpdir).mockReturnValue("/tmp");
  vi.mocked(uuidv4).mockReturnValue("test-uuid" as any);
});

describe("createTempDir", () => {
  it("calls mkdtemp with prefix containing dafny- and uuid", async () => {
    vi.mocked(mkdtemp).mockResolvedValue("/tmp/dafny-test-uuid-abc123");

    const result = await createTempDir();

    expect(mkdtemp).toHaveBeenCalledWith("/tmp/dafny-test-uuid-");
    expect(result).toBe("/tmp/dafny-test-uuid-abc123");
  });
});

describe("removeTempDir", () => {
  it("calls rm with recursive and force options", async () => {
    vi.mocked(rm).mockResolvedValue(undefined);

    await removeTempDir("/tmp/dafny-some-dir");

    expect(rm).toHaveBeenCalledWith("/tmp/dafny-some-dir", {
      recursive: true,
      force: true,
    });
  });
});

describe("cleanupStaleDirs", () => {
  it("removes stale dafny directories older than 30 minutes", async () => {
    const now = Date.now();
    vi.spyOn(Date, "now").mockReturnValue(now);

    vi.mocked(readdir).mockResolvedValue([
      "dafny-old-dir",
      "dafny-fresh-dir",
      "other-dir",
    ] as any);

    vi.mocked(stat)
      .mockResolvedValueOnce({
        mtimeMs: now - 31 * 60 * 1000, // 31 min ago - stale
      } as any)
      .mockResolvedValueOnce({
        mtimeMs: now - 5 * 60 * 1000, // 5 min ago - fresh
      } as any);

    vi.mocked(rm).mockResolvedValue(undefined);

    const cleaned = await cleanupStaleDirs();

    expect(cleaned).toBe(1);
    expect(stat).toHaveBeenCalledTimes(2); // only dafny- prefixed entries
    expect(rm).toHaveBeenCalledWith("/tmp/dafny-old-dir", {
      recursive: true,
      force: true,
    });
    // Should not have removed the fresh one
    expect(rm).not.toHaveBeenCalledWith("/tmp/dafny-fresh-dir", {
      recursive: true,
      force: true,
    });
  });

  it("filters entries by dafny- prefix only", async () => {
    vi.mocked(readdir).mockResolvedValue([
      "not-dafny",
      "something-else",
    ] as any);

    const cleaned = await cleanupStaleDirs();

    expect(cleaned).toBe(0);
    expect(stat).not.toHaveBeenCalled();
  });

  it("handles concurrent removal when stat throws", async () => {
    vi.mocked(readdir).mockResolvedValue(["dafny-gone-dir"] as any);
    vi.mocked(stat).mockRejectedValue(new Error("ENOENT"));

    const cleaned = await cleanupStaleDirs();

    expect(cleaned).toBe(0);
    expect(rm).not.toHaveBeenCalled();
  });
});
