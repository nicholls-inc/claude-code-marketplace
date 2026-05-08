import { mkdtemp, rm, readdir, stat } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { v4 as uuidv4 } from "uuid";

const DAFNY_TMP_PREFIX = "dafny-";
const STALE_THRESHOLD_MS = 30 * 60 * 1000; // 30 minutes

// Prefixes the cleanup sweep recognises. Each tool family registers its own
// so /dafny_cleanup can remove stale dirs across both Dafny and Lean runs
// without leaking implementation knowledge into individual tools.
const TMP_PREFIXES = [DAFNY_TMP_PREFIX, "lean-"];

export async function createTempDir(prefix?: string): Promise<string> {
  const tag = prefix ?? DAFNY_TMP_PREFIX;
  const dir = await mkdtemp(join(tmpdir(), `${tag}${uuidv4()}-`));
  return dir;
}

export async function removeTempDir(dir: string): Promise<void> {
  await rm(dir, { recursive: true, force: true });
}

export async function cleanupStaleDirs(): Promise<number> {
  const base = tmpdir();
  const entries = await readdir(base);
  const now = Date.now();
  let cleaned = 0;

  for (const entry of entries) {
    if (!TMP_PREFIXES.some((p) => entry.startsWith(p))) continue;

    const fullPath = join(base, entry);
    try {
      const stats = await stat(fullPath);
      if (now - stats.mtimeMs > STALE_THRESHOLD_MS) {
        await rm(fullPath, { recursive: true, force: true });
        cleaned++;
      }
    } catch {
      // Entry may have been removed concurrently
    }
  }

  return cleaned;
}
