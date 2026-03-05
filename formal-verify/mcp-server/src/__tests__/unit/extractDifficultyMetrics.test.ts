import { describe, it, expect } from "vitest";
import { extractDifficultyMetrics } from "../../tools/verify.js";

describe("extractDifficultyMetrics", () => {
  describe("solver time parsing", () => {
    it("parses solver time from 'finished in X.XXs' pattern", () => {
      const result = extractDifficultyMetrics(
        "",
        "Dafny program verifier finished in 1.23s"
      );
      expect(result.solverTimeMs).toBe(1230);
    });

    it("parses integer seconds", () => {
      const result = extractDifficultyMetrics(
        "",
        "finished in 5s"
      );
      expect(result.solverTimeMs).toBe(5000);
    });

    it("returns null when no time pattern found", () => {
      const result = extractDifficultyMetrics(
        "",
        "Dafny program verifier finished with 1 verified, 0 errors"
      );
      expect(result.solverTimeMs).toBeNull();
    });
  });

  describe("resource count parsing", () => {
    it("parses resource count from output", () => {
      const result = extractDifficultyMetrics(
        "",
        "resource count: 12345"
      );
      expect(result.resourceCount).toBe(12345);
    });

    it("parses case-insensitive resource count", () => {
      const result = extractDifficultyMetrics(
        "",
        "Resource Count: 999"
      );
      expect(result.resourceCount).toBe(999);
    });

    it("returns null when no resource count found", () => {
      const result = extractDifficultyMetrics("", "no resources here");
      expect(result.resourceCount).toBeNull();
    });
  });

  describe("proof hint counting", () => {
    it("counts assert statements", () => {
      const source = `
method Foo() {
  assert x > 0;
  assert y < 10;
}`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.proofHintCount).toBe(2);
    });

    it("counts calc blocks", () => {
      const source = `
lemma CalcExample()
{
  calc {
    1 + 1;
    == 2;
  }
}`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.proofHintCount).toBe(1);
    });

    it("counts mixed assert and calc", () => {
      const source = `
method Foo() {
  assert x > 0;
  calc {
    a;
    == b;
  }
  assert y == z;
}`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.proofHintCount).toBe(3);
    });

    it("returns 0 when no proof hints", () => {
      const source = "method Main() {}";
      const result = extractDifficultyMetrics(source, "");
      expect(result.proofHintCount).toBe(0);
    });
  });

  describe("empty lemma body detection", () => {
    it("counts lemmas with empty bodies", () => {
      const source = `lemma Trivial() {}`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.emptyLemmaBodyCount).toBe(1);
    });

    it("counts multiple empty lemmas", () => {
      const source = `
lemma Foo() {}
lemma Bar() {}
lemma Baz() { assert true; }`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.emptyLemmaBodyCount).toBe(2);
    });

    it("does not count lemmas with content", () => {
      const source = `lemma NonTrivial() { assert x > 0; }`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.emptyLemmaBodyCount).toBe(0);
    });

    it("counts lemmas with whitespace-only bodies", () => {
      const source = `lemma Trivial() {   }`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.emptyLemmaBodyCount).toBe(1);
    });
  });

  describe("trivialProof derivation", () => {
    it("is true when no hints and empty lemma bodies exist", () => {
      const source = `lemma Trivial() {}`;
      const result = extractDifficultyMetrics(source, "");
      expect(result.trivialProof).toBe(true);
    });

    it("is true when no hints and solver time < 2000ms", () => {
      const source = "method Main() {}";
      const result = extractDifficultyMetrics(
        source,
        "finished in 0.5s"
      );
      expect(result.trivialProof).toBe(true);
    });

    it("is false when hints are present even with fast time", () => {
      const source = `
method Foo() {
  assert x > 0;
}`;
      const result = extractDifficultyMetrics(
        source,
        "finished in 0.1s"
      );
      expect(result.trivialProof).toBe(false);
    });

    it("is false when no hints and no time info and no empty lemmas", () => {
      const source = "method Main() {}";
      const result = extractDifficultyMetrics(source, "");
      expect(result.trivialProof).toBe(false);
    });

    it("is false when solver time >= 2000ms and no empty lemmas", () => {
      const source = "method Main() {}";
      const result = extractDifficultyMetrics(
        source,
        "finished in 3.0s"
      );
      expect(result.trivialProof).toBe(false);
    });
  });

  describe("edge cases", () => {
    it("handles empty source and empty output", () => {
      const result = extractDifficultyMetrics("", "");
      expect(result.solverTimeMs).toBeNull();
      expect(result.resourceCount).toBeNull();
      expect(result.proofHintCount).toBe(0);
      expect(result.emptyLemmaBodyCount).toBe(0);
      expect(result.trivialProof).toBe(false);
    });
  });
});
