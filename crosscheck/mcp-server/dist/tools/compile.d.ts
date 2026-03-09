type Target = "py" | "go";
interface CompileInput {
    source: string;
    target: Target;
}
interface OutputFile {
    path: string;
    content: string;
}
interface CompileOutput {
    success: boolean;
    errors: string[];
    files: OutputFile[];
    rawOutput: string;
}
export declare const PYTHON_EXCLUDE_FILES: string[];
export declare const PYTHON_STRIP_PATTERNS: RegExp[];
export declare const GO_EXCLUDE_FILES: string[];
export declare const GO_EXCLUDE_DIRS: string[];
export declare const GO_STRIP_PATTERNS: RegExp[];
export declare function stripBoilerplate(content: string, target: Target): string;
export declare function shouldExclude(filePath: string, target: Target): boolean;
export declare function collectFiles(dir: string, base: string, target: Target): Promise<OutputFile[]>;
export declare function dafnyCompile(input: CompileInput): Promise<CompileOutput>;
export {};
