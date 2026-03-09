import { mkdtemp, rm, readdir, stat } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { v4 as uuidv4 } from "uuid";
const DAFNY_TMP_PREFIX = "dafny-";
const STALE_THRESHOLD_MS = 30 * 60 * 1000; // 30 minutes
export async function createTempDir() {
    const dir = await mkdtemp(join(tmpdir(), `${DAFNY_TMP_PREFIX}${uuidv4()}-`));
    return dir;
}
export async function removeTempDir(dir) {
    await rm(dir, { recursive: true, force: true });
}
export async function cleanupStaleDirs() {
    const base = tmpdir();
    const entries = await readdir(base);
    const now = Date.now();
    let cleaned = 0;
    for (const entry of entries) {
        if (!entry.startsWith(DAFNY_TMP_PREFIX))
            continue;
        const fullPath = join(base, entry);
        try {
            const stats = await stat(fullPath);
            if (now - stats.mtimeMs > STALE_THRESHOLD_MS) {
                await rm(fullPath, { recursive: true, force: true });
                cleaned++;
            }
        }
        catch {
            // Entry may have been removed concurrently
        }
    }
    return cleaned;
}
