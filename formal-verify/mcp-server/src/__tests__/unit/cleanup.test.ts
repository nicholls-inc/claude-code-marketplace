import { describe, it, expect, vi } from "vitest";

vi.mock("../../tempdir.js", () => ({
  cleanupStaleDirs: vi.fn(),
}));

import { dafnyCleanup } from "../../tools/cleanup.js";
import { cleanupStaleDirs } from "../../tempdir.js";

const mockCleanup = vi.mocked(cleanupStaleDirs);

describe("dafnyCleanup", () => {
  it("returns { cleaned: 0 } when no stale dirs found", async () => {
    mockCleanup.mockResolvedValue(0);
    const result = await dafnyCleanup();
    expect(result).toEqual({ cleaned: 0 });
    expect(mockCleanup).toHaveBeenCalledOnce();
  });

  it("returns { cleaned: 3 } when 3 stale dirs cleaned", async () => {
    mockCleanup.mockResolvedValue(3);
    const result = await dafnyCleanup();
    expect(result).toEqual({ cleaned: 3 });
    expect(mockCleanup).toHaveBeenCalled();
  });
});
