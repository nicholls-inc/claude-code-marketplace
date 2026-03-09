interface VerifyInput {
    source: string;
}
export interface DifficultyMetrics {
    solverTimeMs: number | null;
    resourceCount: number | null;
    proofHintCount: number;
    emptyLemmaBodyCount: number;
    trivialProof: boolean;
}
interface VerifyOutput {
    success: boolean;
    errors: string[];
    warnings: string[];
    rawOutput: string;
    difficulty: DifficultyMetrics;
}
export declare function extractDifficultyMetrics(source: string, rawOutput: string): DifficultyMetrics;
export declare function parseDafnyOutput(stdout: string, stderr: string): {
    errors: string[];
    warnings: string[];
};
export declare function dafnyVerify(input: VerifyInput): Promise<VerifyOutput>;
export {};
