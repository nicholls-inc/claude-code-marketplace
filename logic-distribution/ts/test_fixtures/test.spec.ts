/** Test: Test file → TEST_CODE */
import { describe, it, expect } from "vitest";
import { formatCurrency } from "./utils";

describe("formatCurrency", () => {
  it("formats USD correctly", () => {
    expect(formatCurrency(42.5)).toBe("$42.50");
  });

  it("formats EUR correctly", () => {
    expect(formatCurrency(42.5, "EUR")).toBe("€42.50");
  });
});
