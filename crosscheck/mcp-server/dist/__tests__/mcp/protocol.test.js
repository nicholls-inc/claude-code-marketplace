import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
vi.mock("../../docker.js", () => ({
    getDockerImage: vi.fn(() => "test-image:latest"),
    runDafny: vi.fn(async () => ({
        exitCode: 0,
        stdout: "Dafny program verifier finished with 1 verified, 0 errors\n",
        stderr: "",
        timedOut: false,
    })),
}));
vi.mock("../../tempdir.js", () => ({
    createTempDir: vi.fn(async () => "/tmp/dafny-test-mock"),
    removeTempDir: vi.fn(async () => { }),
    cleanupStaleDirs: vi.fn(async () => 0),
}));
vi.mock("node:fs/promises", async (importOriginal) => {
    const orig = await importOriginal();
    return {
        ...orig,
        writeFile: vi.fn(async () => { }),
        readdir: vi.fn(async () => []),
        readFile: vi.fn(async () => ""),
    };
});
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import { createServer } from "../../index.js";
describe("MCP Protocol", () => {
    let client;
    beforeEach(async () => {
        const server = createServer();
        const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();
        await server.connect(serverTransport);
        client = new Client({ name: "test-client", version: "1.0.0" });
        await client.connect(clientTransport);
    });
    afterEach(async () => {
        await client.close();
    });
    describe("listTools", () => {
        it("returns 3 tools: dafny_verify, dafny_compile, dafny_cleanup", async () => {
            const result = await client.listTools();
            const names = result.tools.map((t) => t.name).sort();
            expect(names).toEqual(["dafny_cleanup", "dafny_compile", "dafny_verify"]);
            expect(result.tools).toHaveLength(3);
        });
    });
    describe("dafny_verify", () => {
        it("returns structured JSON with success/errors/warnings/rawOutput", async () => {
            const result = await client.callTool({
                name: "dafny_verify",
                arguments: { source: "method Main() {}" },
            });
            expect(result.content).toHaveLength(1);
            const item = result.content[0];
            expect(item.type).toBe("text");
            const parsed = JSON.parse(item.text);
            expect(parsed).toHaveProperty("success");
            expect(parsed).toHaveProperty("errors");
            expect(parsed).toHaveProperty("warnings");
            expect(parsed).toHaveProperty("rawOutput");
            expect(parsed).toHaveProperty("difficulty");
            expect(parsed.success).toBe(true);
            expect(parsed.errors).toEqual([]);
            expect(parsed.difficulty).toEqual(expect.objectContaining({
                proofHintCount: expect.any(Number),
                emptyLemmaBodyCount: expect.any(Number),
                trivialProof: expect.any(Boolean),
            }));
        });
    });
    describe("dafny_compile", () => {
        it("returns structured JSON with success/errors/files/rawOutput", async () => {
            const result = await client.callTool({
                name: "dafny_compile",
                arguments: { source: "method Main() {}", target: "py" },
            });
            expect(result.content).toHaveLength(1);
            const item = result.content[0];
            expect(item.type).toBe("text");
            const parsed = JSON.parse(item.text);
            expect(parsed).toHaveProperty("success");
            expect(parsed).toHaveProperty("errors");
            expect(parsed).toHaveProperty("files");
            expect(parsed).toHaveProperty("rawOutput");
            expect(parsed.success).toBe(true);
            expect(parsed.errors).toEqual([]);
            expect(Array.isArray(parsed.files)).toBe(true);
        });
    });
    describe("dafny_cleanup", () => {
        it("returns structured JSON with cleaned count", async () => {
            const result = await client.callTool({
                name: "dafny_cleanup",
                arguments: {},
            });
            expect(result.content).toHaveLength(1);
            const item = result.content[0];
            expect(item.type).toBe("text");
            const parsed = JSON.parse(item.text);
            expect(parsed).toHaveProperty("cleaned");
            expect(parsed.cleaned).toBe(0);
        });
    });
    describe("error handling", () => {
        it("returns error for invalid tool name", async () => {
            const result = await client.callTool({
                name: "nonexistent_tool",
                arguments: {},
            });
            expect(result.isError).toBe(true);
        });
        it("returns error when dafny_verify is called without required source param", async () => {
            const result = await client.callTool({
                name: "dafny_verify",
                arguments: {},
            });
            // MCP SDK returns isError flag or throws for validation errors
            // The server should reject missing required params
            const item = result.content[0];
            expect(result.isError).toBe(true);
        });
        it("returns error when dafny_compile is called with invalid target enum", async () => {
            const result = await client.callTool({
                name: "dafny_compile",
                arguments: { source: "method Main() {}", target: "invalid" },
            });
            const item = result.content[0];
            expect(result.isError).toBe(true);
        });
    });
});
