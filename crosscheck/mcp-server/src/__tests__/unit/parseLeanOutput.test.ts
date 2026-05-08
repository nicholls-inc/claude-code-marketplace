import { describe, it, expect } from "vitest";
import { parseLeanOutput, classifyLeanFailure } from "../../tools/leanCheck.js";

describe("parseLeanOutput", () => {
  it("returns empty arrays for empty input", () => {
    const result = parseLeanOutput("", "");
    expect(result).toEqual({ errors: [], warnings: [], sorries: [] });
  });

  it("classifies file:line:col error: lines as errors", () => {
    const result = parseLeanOutput(
      "Crosscheck/Program.lean:10:5: error: unknown identifier 'foo'",
      ""
    );
    expect(result.errors).toEqual([
      "Crosscheck/Program.lean:10:5: error: unknown identifier 'foo'",
    ]);
    expect(result.warnings).toEqual([]);
    expect(result.sorries).toEqual([]);
  });

  it("classifies file:line:col warning: lines as warnings", () => {
    const result = parseLeanOutput(
      "Crosscheck/Program.lean:5:0: warning: unused variable 'x'",
      ""
    );
    expect(result.warnings).toEqual([
      "Crosscheck/Program.lean:5:0: warning: unused variable 'x'",
    ]);
    expect(result.errors).toEqual([]);
  });

  it("routes 'declaration uses sorry' warnings into sorries, not warnings", () => {
    const result = parseLeanOutput(
      "Crosscheck/Program.lean:7:0: warning: declaration uses 'sorry'",
      ""
    );
    expect(result.sorries).toHaveLength(1);
    expect(result.warnings).toEqual([]);
    expect(result.errors).toEqual([]);
  });

  it("strips ANSI colour codes from messages", () => {
    const result = parseLeanOutput(
      "\x1b[31mCrosscheck/Program.lean:1:0: error: bad\x1b[0m",
      ""
    );
    expect(result.errors).toEqual(["Crosscheck/Program.lean:1:0: error: bad"]);
  });

  it("recognises top-level 'error:' messages from lake", () => {
    const result = parseLeanOutput("error: build failed", "");
    expect(result.errors).toEqual(["error: build failed"]);
  });

  it("includes stderr lines in parsing", () => {
    const result = parseLeanOutput("", "stderr error: oops");
    expect(result.errors).toEqual(["stderr error: oops"]);
  });

  it("filters out whitespace-only lines", () => {
    const result = parseLeanOutput("   \n\t\n  ", "");
    expect(result).toEqual({ errors: [], warnings: [], sorries: [] });
  });
});

describe("classifyLeanFailure", () => {
  it("returns success when exitCode is 0 and no errors", () => {
    expect(classifyLeanFailure(0, [])).toBe("success");
  });

  it("returns parse-error when an error mentions 'unexpected token'", () => {
    expect(
      classifyLeanFailure(1, ["Crosscheck/Program.lean:3:0: error: unexpected token 'def'"])
    ).toBe("parse-error");
  });

  it("returns parse-error when an error mentions 'expected'", () => {
    expect(
      classifyLeanFailure(1, ["Crosscheck/Program.lean:3:0: error: expected ':='"])
    ).toBe("parse-error");
  });

  it("returns typecheck-error when an error mentions type mismatch", () => {
    expect(
      classifyLeanFailure(1, ["Crosscheck/Program.lean:9:0: error: type mismatch"])
    ).toBe("typecheck-error");
  });

  it("returns typecheck-error for failed-to-synthesize", () => {
    expect(
      classifyLeanFailure(1, ["error: failed to synthesize Decidable (1 = 1)"])
    ).toBe("typecheck-error");
  });

  it("falls back to build-error for unrecognised non-zero exits", () => {
    expect(classifyLeanFailure(1, ["error: build failed"])).toBe("build-error");
  });

  it("treats parse-error as winning over typecheck-error when both present", () => {
    expect(
      classifyLeanFailure(1, [
        "error: type mismatch",
        "error: unexpected token 'foo'",
      ])
    ).toBe("parse-error");
  });
});
