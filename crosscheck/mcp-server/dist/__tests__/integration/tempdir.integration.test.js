import { describe, it, expect, afterEach } from "vitest";
import { mkdtemp, stat, utimes } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { createTempDir, removeTempDir, cleanupStaleDirs, } from "../../tempdir.js";
describe("tempdir integration", () => {
    const dirsToCleanup = [];
    afterEach(async () => {
        for (const dir of dirsToCleanup) {
            try {
                await removeTempDir(dir);
            }
            catch {
                // Already removed by the test
            }
        }
        dirsToCleanup.length = 0;
    });
    it("createTempDir creates a real directory with dafny- prefix", async () => {
        const dir = await createTempDir();
        dirsToCleanup.push(dir);
        const stats = await stat(dir);
        expect(stats.isDirectory()).toBe(true);
        expect(dir).toContain("dafny-");
    });
    it("removeTempDir actually deletes the directory", async () => {
        const dir = await createTempDir();
        // Verify it exists first
        const stats = await stat(dir);
        expect(stats.isDirectory()).toBe(true);
        await removeTempDir(dir);
        // Verify it is gone
        await expect(stat(dir)).rejects.toThrow();
    });
    it("cleanupStaleDirs removes dirs with old mtime", async () => {
        const base = tmpdir();
        const dir = await mkdtemp(join(base, "dafny-test-"));
        dirsToCleanup.push(dir);
        // Set mtime to 31 minutes ago
        const oldTime = new Date(Date.now() - 31 * 60 * 1000);
        await utimes(dir, oldTime, oldTime);
        const cleaned = await cleanupStaleDirs();
        expect(cleaned).toBeGreaterThanOrEqual(1);
        // Verify dir is gone
        await expect(stat(dir)).rejects.toThrow();
    });
    it("cleanupStaleDirs leaves fresh dirs alone", async () => {
        const base = tmpdir();
        const dir = await mkdtemp(join(base, "dafny-test-"));
        dirsToCleanup.push(dir);
        // Don't change mtime - it's fresh
        const cleaned = await cleanupStaleDirs();
        // The fresh dir should still exist
        const stats = await stat(dir);
        expect(stats.isDirectory()).toBe(true);
    });
});
