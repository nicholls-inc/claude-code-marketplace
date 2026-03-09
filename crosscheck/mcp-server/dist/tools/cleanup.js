import { cleanupStaleDirs } from "../tempdir.js";
export async function dafnyCleanup() {
    const cleaned = await cleanupStaleDirs();
    return { cleaned };
}
