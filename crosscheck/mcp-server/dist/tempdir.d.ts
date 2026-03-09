export declare function createTempDir(): Promise<string>;
export declare function removeTempDir(dir: string): Promise<void>;
export declare function cleanupStaleDirs(): Promise<number>;
