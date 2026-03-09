import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { shouldExclude } from "../../tools/compile.js";
describe("shouldExclude property tests", () => {
    describe("known excludes always return true", () => {
        it("paths containing /dafny/ are excluded for go target", () => {
            fc.assert(fc.property(fc.stringMatching(/^[a-z]{1,8}$/), fc.stringMatching(/^[a-z]{1,8}\.go$/), (prefix, suffix) => {
                const path = `${prefix}/dafny/${suffix}`;
                expect(shouldExclude(path, "go")).toBe(true);
            }));
        });
        it("paths containing /System_/ are excluded for go target", () => {
            fc.assert(fc.property(fc.stringMatching(/^[a-z]{1,8}$/), fc.stringMatching(/^[a-z]{1,8}\.go$/), (prefix, suffix) => {
                const path = `${prefix}/System_/${suffix}`;
                expect(shouldExclude(path, "go")).toBe(true);
            }));
        });
        it("paths containing _dafny.py are excluded for py target", () => {
            fc.assert(fc.property(fc.stringMatching(/^[a-z]{1,8}$/), (prefix) => {
                const path = `${prefix}/_dafny.py`;
                expect(shouldExclude(path, "py")).toBe(true);
            }));
        });
        it("paths containing __pycache__ are excluded for py target", () => {
            fc.assert(fc.property(fc.stringMatching(/^[a-z]{1,8}$/), (prefix) => {
                const path = `${prefix}/__pycache__/module.py`;
                expect(shouldExclude(path, "py")).toBe(true);
            }));
        });
    });
    describe("safe filenames are never excluded", () => {
        // Generate filenames that cannot match any exclude pattern:
        // lowercase letters only, 1-8 chars, with .py or .go extension
        const safePyFilename = fc
            .stringMatching(/^[a-z]{1,8}$/)
            .map((name) => `${name}.py`);
        const safeGoFilename = fc
            .stringMatching(/^[a-z]{1,8}$/)
            .map((name) => `${name}.go`);
        it("safe Python filenames are not excluded for py target", () => {
            fc.assert(fc.property(safePyFilename, (filename) => {
                expect(shouldExclude(filename, "py")).toBe(false);
            }));
        });
        it("safe Go filenames are not excluded for go target", () => {
            fc.assert(fc.property(safeGoFilename, (filename) => {
                expect(shouldExclude(filename, "go")).toBe(false);
            }));
        });
    });
    describe("target isolation", () => {
        it("Python exclude filenames do not affect go target (basename check)", () => {
            // _dafny.py basename is not in GO_EXCLUDE_FILES and the path
            // doesn't contain /dafny/ or /System_/
            expect(shouldExclude("_dafny.py", "go")).toBe(false);
            expect(shouldExclude("__pycache__", "go")).toBe(false);
        });
        it("Go exclude filenames do not affect py target", () => {
            // dafny.go and System_.go basenames are not in PYTHON_EXCLUDE_FILES
            // and the paths don't include "_dafny.py" or "__pycache__"
            expect(shouldExclude("dafny.go", "py")).toBe(false);
            expect(shouldExclude("System_.go", "py")).toBe(false);
        });
        it("arbitrary safe filenames are not excluded regardless of target", () => {
            const safeFilename = fc
                .stringMatching(/^[a-z]{1,8}$/)
                .map((name) => `${name}.txt`);
            const targetArb = fc.constantFrom("py", "go");
            fc.assert(fc.property(safeFilename, targetArb, (filename, target) => {
                expect(shouldExclude(filename, target)).toBe(false);
            }));
        });
    });
});
