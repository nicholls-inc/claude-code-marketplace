import { cleanupStaleDirs } from "../tempdir.js";

interface CleanupOutput {
  cleaned: number;
}

export async function dafnyCleanup(): Promise<CleanupOutput> {
  const cleaned = await cleanupStaleDirs();
  return { cleaned };
}
