import { describe, it, expect } from "vitest";
import { parseDafnyOutput } from "../../tools/verify.js";

describe("parseDafnyOutput", () => {
  it("returns empty arrays for empty input", () => {
    const result = parseDafnyOutput("", "");
    expect(result).toEqual({ errors: [], warnings: [] });
  });

  it("classifies a line containing 'Error' as an error", () => {
    const result = parseDafnyOutput("program.dfy(3,4): Error: something went wrong", "");
    expect(result.errors).toEqual(["program.dfy(3,4): Error: something went wrong"]);
    expect(result.warnings).toEqual([]);
  });

  it("classifies a line containing 'Warning' as a warning", () => {
    const result = parseDafnyOutput("program.dfy(5,0): Warning: unused variable", "");
    expect(result.warnings).toEqual(["program.dfy(5,0): Warning: unused variable"]);
    expect(result.errors).toEqual([]);
  });

  it("does NOT classify 'Dafny program verifier' summary line as an error", () => {
    const result = parseDafnyOutput(
      "Dafny program verifier finished with 0 verified, 1 error",
      ""
    );
    expect(result.errors).toEqual([]);
    expect(result.warnings).toEqual([]);
  });

  it("matches 'error' case-insensitively", () => {
    const lower = parseDafnyOutput("some error here", "");
    expect(lower.errors).toEqual(["some error here"]);

    const upper = parseDafnyOutput("some ERROR here", "");
    expect(upper.errors).toEqual(["some ERROR here"]);
  });

  it("includes stderr lines in parsing", () => {
    const result = parseDafnyOutput("", "stderr Error: bad thing");
    expect(result.errors).toEqual(["stderr Error: bad thing"]);
  });

  it("filters out whitespace-only lines", () => {
    const result = parseDafnyOutput("   \n\t\n  ", "");
    expect(result).toEqual({ errors: [], warnings: [] });
  });

  it("classifies a line with both 'Error' and 'Warning' as an error (Error check first)", () => {
    const result = parseDafnyOutput("Error: this is also a Warning", "");
    expect(result.errors).toEqual(["Error: this is also a Warning"]);
    expect(result.warnings).toEqual([]);
  });
});
