import { describe, it, expect } from "vitest";
import { shouldExclude } from "../../tools/compile.js";
describe("shouldExclude", () => {
    describe("Python target", () => {
        it("excludes _dafny.py", () => {
            expect(shouldExclude("_dafny.py", "py")).toBe(true);
        });
        it("excludes __pycache__", () => {
            expect(shouldExclude("__pycache__", "py")).toBe(true);
        });
        it("excludes paths containing __pycache__", () => {
            expect(shouldExclude("some/path/__pycache__/foo", "py")).toBe(true);
        });
        it("does not exclude main.py", () => {
            expect(shouldExclude("main.py", "py")).toBe(false);
        });
    });
    describe("Go target", () => {
        it("excludes dafny.go", () => {
            expect(shouldExclude("dafny.go", "go")).toBe(true);
        });
        it("excludes System_.go", () => {
            expect(shouldExclude("System_.go", "go")).toBe(true);
        });
        it("excludes paths containing /dafny/ directory", () => {
            expect(shouldExclude("some/dafny/file.go", "go")).toBe(true);
        });
        it("excludes paths containing /System_/ directory", () => {
            expect(shouldExclude("some/System_/file.go", "go")).toBe(true);
        });
        it("does not exclude main.go", () => {
            expect(shouldExclude("main.go", "go")).toBe(false);
        });
    });
    it("returns false for non-matching target", () => {
        // A file that would be excluded for "py" is not excluded for "go"
        expect(shouldExclude("_dafny.py", "go")).toBe(false);
        // A file that would be excluded for "go" is not excluded for "py"
        expect(shouldExclude("dafny.go", "py")).toBe(false);
    });
});
