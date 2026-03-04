import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { parseDafnyOutput } from "../../tools/verify.js";

describe("parseDafnyOutput property tests", () => {
  it("every string in errors contains 'Error' (case-insensitive)", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (stdout, stderr) => {
        const { errors } = parseDafnyOutput(stdout, stderr);
        for (const err of errors) {
          expect(err.toLowerCase()).toContain("error");
        }
      })
    );
  });

  it("every string in warnings contains 'Warning' (case-insensitive)", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (stdout, stderr) => {
        const { warnings } = parseDafnyOutput(stdout, stderr);
        for (const warn of warnings) {
          expect(warn.toLowerCase()).toContain("warning");
        }
      })
    );
  });

  it("no line appears in both errors and warnings", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (stdout, stderr) => {
        const { errors, warnings } = parseDafnyOutput(stdout, stderr);
        const errorsSet = new Set(errors);
        for (const warn of warnings) {
          expect(errorsSet.has(warn)).toBe(false);
        }
      })
    );
  });

  it("lines with both Error and Warning go to errors only (Error takes precedence)", () => {
    const mixedLineArb = fc
      .stringMatching(/^[a-zA-Z0-9 ]{0,20}$/)
      .map((prefix) => `${prefix} Error and Warning combined`);

    fc.assert(
      fc.property(mixedLineArb, (line) => {
        const { errors, warnings } = parseDafnyOutput(line, "");
        expect(errors.some((e) => e.includes("Error and Warning combined"))).toBe(true);
        expect(warnings.some((w) => w.includes("Error and Warning combined"))).toBe(false);
      })
    );
  });

  it("'Dafny program verifier' lines are excluded from errors", () => {
    const verifierSummary = fc.constantFrom(
      "Dafny program verifier finished with 0 verified, 1 error",
      "Dafny program verifier finished with 3 verified, 0 errors",
      "Dafny program verifier finished with 1 verified, 2 errors"
    );

    fc.assert(
      fc.property(verifierSummary, (line) => {
        const { errors } = parseDafnyOutput(line, "");
        expect(errors).toEqual([]);
      })
    );
  });

  it("empty input gives empty arrays", () => {
    const { errors, warnings } = parseDafnyOutput("", "");
    expect(errors).toEqual([]);
    expect(warnings).toEqual([]);
  });

  it("structured line arrays produce correct categorization", () => {
    const lineArb = fc.oneof(
      fc.constant("some Error line"),
      fc.constant("some Warning line"),
      fc.constant("clean line with no markers")
    );

    fc.assert(
      fc.property(
        fc.array(lineArb, { minLength: 0, maxLength: 20 }),
        (lines) => {
          const input = lines.join("\n");
          const { errors, warnings } = parseDafnyOutput(input, "");

          const expectedErrors = lines
            .filter((l) => /Error/i.test(l))
            .map((l) => l.trim());
          const expectedWarnings = lines
            .filter((l) => /Warning/i.test(l) && !/Error/i.test(l))
            .map((l) => l.trim());

          expect(errors).toEqual(expectedErrors);
          expect(warnings).toEqual(expectedWarnings);
        }
      )
    );
  });

  it("total categorized lines never exceed total non-empty input lines", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (stdout, stderr) => {
        const combined = stdout + "\n" + stderr;
        const nonEmptyLines = combined.split("\n").filter((l) => l.trim()).length;
        const { errors, warnings } = parseDafnyOutput(stdout, stderr);
        expect(errors.length + warnings.length).toBeLessThanOrEqual(nonEmptyLines);
      })
    );
  });
});
