import { describe, it, expect } from "vitest";
import { dafnyCompile } from "../../tools/compile.js";
describe.skipIf(!process.env.RUN_E2E)("dafny_compile E2E", () => {
    const SIMPLE_PROGRAM = `
method Main()
{
  print "Hello\\n";
}`;
    it("compiles to Python and produces .py files", async () => {
        const result = await dafnyCompile({
            source: SIMPLE_PROGRAM,
            target: "py",
        });
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
        expect(result.files.length).toBeGreaterThan(0);
        for (const file of result.files) {
            expect(file.path).toMatch(/\.py$/);
            // Should not contain dafny runtime imports after stripping
            expect(file.content).not.toMatch(/from _dafny import/);
            expect(file.content).not.toMatch(/import _dafny/);
        }
    }, 180_000);
    it("compiles to Go and produces .go files", async () => {
        const result = await dafnyCompile({
            source: SIMPLE_PROGRAM,
            target: "go",
        });
        expect(result.success).toBe(true);
        expect(result.errors).toHaveLength(0);
        expect(result.files.length).toBeGreaterThan(0);
        for (const file of result.files) {
            expect(file.path).toMatch(/\.go$/);
        }
    }, 180_000);
    it("fails on invalid source", async () => {
        const result = await dafnyCompile({
            source: `method Broken( {`,
            target: "py",
        });
        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
    }, 180_000);
    it("strips boilerplate from real Dafny output", async () => {
        const result = await dafnyCompile({
            source: SIMPLE_PROGRAM,
            target: "py",
        });
        if (result.success && result.files.length > 0) {
            for (const file of result.files) {
                expect(file.content).not.toMatch(/from _System import/);
                expect(file.content).not.toMatch(/import _System/);
            }
        }
    }, 180_000);
});
