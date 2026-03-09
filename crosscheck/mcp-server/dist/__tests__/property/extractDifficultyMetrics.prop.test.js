import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { extractDifficultyMetrics } from "../../tools/verify.js";
describe("extractDifficultyMetrics property tests", () => {
    it("counts are always non-negative", () => {
        fc.assert(fc.property(fc.string(), fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            expect(result.proofHintCount).toBeGreaterThanOrEqual(0);
            expect(result.emptyLemmaBodyCount).toBeGreaterThanOrEqual(0);
        }));
    });
    it("solverTimeMs is null or >= 0", () => {
        fc.assert(fc.property(fc.string(), fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            if (result.solverTimeMs !== null) {
                expect(result.solverTimeMs).toBeGreaterThanOrEqual(0);
            }
        }));
    });
    it("resourceCount is null or >= 0", () => {
        fc.assert(fc.property(fc.string(), fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            if (result.resourceCount !== null) {
                expect(result.resourceCount).toBeGreaterThanOrEqual(0);
            }
        }));
    });
    it("no lemma keyword in source means emptyLemmaBodyCount === 0", () => {
        const noLemmaArb = fc
            .string()
            .filter((s) => !s.includes("lemma"));
        fc.assert(fc.property(noLemmaArb, fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            expect(result.emptyLemmaBodyCount).toBe(0);
        }));
    });
    it("trivialProof is consistent: true only when proofHintCount === 0", () => {
        fc.assert(fc.property(fc.string(), fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            if (result.trivialProof) {
                expect(result.proofHintCount).toBe(0);
            }
        }));
    });
    it("trivialProof is false when proofHintCount > 0", () => {
        // Source with at least one assert statement
        const sourceWithAssert = fc
            .string()
            .map((s) => `  assert something;\n${s}`);
        fc.assert(fc.property(sourceWithAssert, fc.string(), (source, output) => {
            const result = extractDifficultyMetrics(source, output);
            if (result.proofHintCount > 0) {
                expect(result.trivialProof).toBe(false);
            }
        }));
    });
});
