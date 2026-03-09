import { describe, it, expect } from "vitest";
import { dafnyVerify } from "../../tools/verify.js";
describe.skipIf(!process.env.RUN_E2E)("dafny_verify E2E", () => {
    // 180s timeout per test — Dafny via Docker can be slow on first run
    it("verifies a valid program", async () => {
        const result = await dafnyVerify({
            source: `method Main() ensures true { }`,
        });
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
    }, 180_000);
    it("fails on invalid postcondition", async () => {
        const result = await dafnyVerify({
            source: `method Foo() ensures false { }`,
        });
        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
    }, 180_000);
    it("reports syntax errors", async () => {
        const result = await dafnyVerify({
            source: `method Broken( {`,
        });
        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
    }, 180_000);
    it("verifies complex program with pre/postconditions", async () => {
        const result = await dafnyVerify({
            source: `
method Abs(x: int) returns (y: int)
  ensures y >= 0
  ensures y == x || y == -x
{
  if x < 0 {
    y := -x;
  } else {
    y := x;
  }
}`,
        });
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
    }, 180_000);
});
