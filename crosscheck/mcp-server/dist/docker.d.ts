interface DockerResult {
    exitCode: number;
    stdout: string;
    stderr: string;
    timedOut: boolean;
}
export declare function getDockerImage(): string;
export declare function runDafny(tempDir: string, args: string[]): Promise<DockerResult>;
export {};
