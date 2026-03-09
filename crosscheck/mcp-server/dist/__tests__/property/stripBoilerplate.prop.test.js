import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { stripBoilerplate } from "../../tools/compile.js";
const targetArb = fc.constantFrom("py", "go");
describe("stripBoilerplate property tests", () => {
    it("is idempotent: applying twice yields the same result as once", () => {
        fc.assert(fc.property(fc.string(), targetArb, (s, target) => {
            const once = stripBoilerplate(s, target);
            const twice = stripBoilerplate(once, target);
            expect(twice).toBe(once);
        }));
    });
    it("output never contains Python boilerplate patterns when target is py", () => {
        fc.assert(fc.property(fc.string(), (s) => {
            const result = stripBoilerplate(s, "py");
            expect(result).not.toMatch(/^from _dafny import.*$/m);
            expect(result).not.toMatch(/^import _dafny.*$/m);
            expect(result).not.toMatch(/^import _System.*$/m);
            expect(result).not.toMatch(/^from _System import.*$/m);
        }));
    });
    it("output never contains Go boilerplate patterns when target is go", () => {
        fc.assert(fc.property(fc.string(), (s) => {
            const result = stripBoilerplate(s, "go");
            expect(result).not.toMatch(/^\s*_dafny\s+"[^"]*dafny".*$/m);
            expect(result).not.toMatch(/^\s*_System\s+"[^"]*System_".*$/m);
        }));
    });
    it("output always ends with a newline for any non-empty input", () => {
        fc.assert(fc.property(fc.string({ minLength: 1 }), targetArb, (s, target) => {
            const result = stripBoilerplate(s, target);
            expect(result.endsWith("\n")).toBe(true);
        }));
    });
    it("output never contains three or more consecutive newlines", () => {
        fc.assert(fc.property(fc.string(), targetArb, (s, target) => {
            const result = stripBoilerplate(s, target);
            expect(result).not.toMatch(/\n{3,}/);
        }));
    });
    it("preserves clean content that has no boilerplate patterns (trimmed + newline)", () => {
        // Generate strings that definitely don't contain any boilerplate patterns
        const cleanContent = fc
            .array(fc.stringMatching(/^[a-zA-Z0-9 ]{0,20}$/), {
            minLength: 1,
            maxLength: 5,
        })
            .map((lines) => lines.join("\n"));
        fc.assert(fc.property(cleanContent, targetArb, (s, target) => {
            const result = stripBoilerplate(s, target);
            // After stripping (no-op), collapsing newlines, and trim + "\n"
            const expected = s.replace(/\n{3,}/g, "\n\n").trim() + "\n";
            expect(result).toBe(expected);
        }));
    });
});
